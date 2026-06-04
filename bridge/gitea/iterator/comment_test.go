package iterator

import (
	"context"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
)

// TestCommentIteratorPaginates documents finding #4: lastPage is set to true
// unconditionally after the first successful API call, so comments beyond the
// first page are silently dropped.
//
// After the fix: all pages should be iterated.
func TestCommentIteratorPaginates(t *testing.T) {
	const capacity = 2
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "Test",
			Poster:  &gitea.User{UserName: "u"},
			Created: ts,
		}},
		Comments: []*gitea.Comment{
			{ID: 1, Body: "c1", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			{ID: 2, Body: "c2", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			{ID: 3, Body: "c3", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			{ID: 4, Body: "c4", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	var got []*gitea.Comment
	for iter.NextComment() {
		got = append(got, iter.CommentValue())
	}
	require.NoError(t, iter.Error())

	assert.Len(t, got, len(fa.Comments),
		"all %d comments should be returned across pages (finding #4: lastPage set too early truncates at page 1)",
		len(fa.Comments))
}

// TestCommentIteratorStopsOnPartialPage documents that a partial page
// (len < capacity) implies no more pages — the iterator should not issue an
// extra empty request to confirm. Current heuristic uses `len(items) != 0`,
// which costs one wasted request per issue.
//
// After the fix: 3 comments at capacity 2 → 2 requests (page 1 full, page 2 partial → done).
func TestCommentIteratorStopsOnPartialPage(t *testing.T) {
	const capacity = 2
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
		Comments: []*gitea.Comment{
			{ID: 1, Body: "c1", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			{ID: 2, Body: "c2", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			{ID: 3, Body: "c3", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())
	for iter.NextComment() {
	}
	require.NoError(t, iter.Error())

	assert.Len(t, fa.CommentRequests, 2,
		"a partial last page should end pagination; no extra empty request needed")
}

// TestIteratorNoTrailingCommentCall verifies exact-capacity comment pages do
// not require a trailing empty request to terminate.
func TestIteratorNoTrailingCommentCall(t *testing.T) {
	const capacity = 2
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
		Comments: []*gitea.Comment{
			{ID: 1, Body: "c1", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			{ID: 2, Body: "c2", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())
	for iter.NextComment() {
	}
	require.NoError(t, iter.Error())

	assert.Len(t, fa.CommentRequests, 1,
		"exactly capacity comments should not trigger a trailing empty comment request")
}

// TestCommentIteratorPassesSince documents that fetchComments ignores
// conf.since. An incremental import filters issues by `since` but re-fetches
// every comment on every matched issue, which is wasteful and inconsistent.
//
// After the fix: the `since` query parameter should appear in the API request.
func TestCommentIteratorPassesSince(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
	}
	srv := fa.NewServer(t)

	since := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, since)
	require.True(t, iter.NextIssue())
	iter.NextComment()
	require.NoError(t, iter.Error())

	require.Len(t, fa.CommentRequests, 1)
	capturedSince := fa.CommentRequests[0].URL.Query().Get("since")

	assert.NotEmpty(t, capturedSince,
		"since parameter should be forwarded to ListIssueComments")

	parsed, err := time.Parse(time.RFC3339, capturedSince)
	require.NoError(t, err, "since value %q should be valid RFC3339", capturedSince)
	assert.Equal(t, since.UTC(), parsed.UTC())
}

// TestCommentIteratorResetsBetweenIssues verifies that advancing to a new
// issue invalidates the comment cache so comments for issue N+1 aren't
// returned when CommentValue is queried under issue N's scope.
func TestCommentIteratorResetsBetweenIssues(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "first", Poster: &gitea.User{UserName: "u"}, Created: ts},
			{ID: 2, Index: 2, Title: "second", Poster: &gitea.User{UserName: "u"}, Created: ts},
		},
		CommentsByIssue: map[int64][]*gitea.Comment{
			1: {{ID: 11, Body: "issue-1-a", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts}},
			2: {{ID: 21, Body: "issue-2-a", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts}},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	gotByIssue := map[int64][]string{}
	for iter.NextIssue() {
		idx := iter.IssueValue().Index
		for iter.NextComment() {
			gotByIssue[idx] = append(gotByIssue[idx], iter.CommentValue().Body)
		}
	}
	require.NoError(t, iter.Error())

	assert.Equal(t, []string{"issue-1-a"}, gotByIssue[1])
	assert.Equal(t, []string{"issue-2-a"}, gotByIssue[2])
}
