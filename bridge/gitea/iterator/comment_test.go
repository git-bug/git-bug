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

	// A partial page proves the listing is exhausted; probing one more empty
	// page costs one wasted request per issue.
	assert.Len(t, fa.CommentRequests, 2,
		"a partial last page should end pagination; no extra empty request needed")
}

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
