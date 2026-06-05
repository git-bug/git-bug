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

func TestCommentIteratorReturnsAPIError(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:           "owner",
		Project:         "repo",
		TimelineErrPage: 1,
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	require.NotPanics(t, func() {
		assert.False(t, iter.NextEvent())
	})
	assert.Error(t, iter.Error())
}

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
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				{ID: 1, Type: "comment", Body: "c1", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
				{ID: 2, Type: "comment", Body: "c2", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
				{ID: 3, Type: "comment", Body: "c3", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
				{ID: 4, Type: "comment", Body: "c4", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	var got []*CommentEvent
	for iter.NextEvent() {
		if e, ok := iter.EventValue().(*CommentEvent); ok {
			got = append(got, e)
		}
	}
	require.NoError(t, iter.Error())

	assert.Len(t, got, len(fa.TimelineByIssue[1]),
		"all %d comments should be returned across pages (finding #4: lastPage set too early truncates at page 1)",
		len(fa.TimelineByIssue[1]))
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
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				{ID: 1, Type: "comment", Body: "c1", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
				{ID: 2, Type: "comment", Body: "c2", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
				{ID: 3, Type: "comment", Body: "c3", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())
	for iter.NextEvent() {
	}
	require.NoError(t, iter.Error())

	// A partial page proves the listing is exhausted; probing one more empty
	// page costs one wasted request per issue.
	assert.Len(t, fa.TimelineRequests, 2,
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
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				{ID: 1, Type: "comment", Body: "c1", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
				{ID: 2, Type: "comment", Body: "c2", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())
	for iter.NextEvent() {
	}
	require.NoError(t, iter.Error())

	assert.Len(t, fa.TimelineRequests, 1,
		"exactly capacity comments should not trigger a trailing empty timeline request")
}

func TestCommentIteratorPassesSince(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts,
		}},
	}
	srv := fa.NewServer(t)

	since := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, since)
	require.True(t, iter.NextIssue())
	iter.NextEvent()
	require.NoError(t, iter.Error())

	require.Len(t, fa.TimelineRequests, 1)
	capturedSince := fa.TimelineRequests[0].URL.Query().Get("since")

	assert.NotEmpty(t, capturedSince,
		"since parameter should be forwarded to timeline endpoint")

	parsed, err := time.Parse(time.RFC3339, capturedSince)
	require.NoError(t, err, "since value %q should be valid RFC3339", capturedSince)
	assert.Equal(t, since.UTC(), parsed.UTC())
}

func TestCommentIteratorRespectsSince(t *testing.T) {
	since := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	before := since.Add(-24 * time.Hour)
	after := since.Add(24 * time.Hour)

	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts,
		}},
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {
				{ID: 1, Type: "comment", Body: "old", Poster: &gitea.User{UserName: "u"}, Created: before, Updated: before},
				{ID: 2, Type: "comment", Body: "new", Poster: &gitea.User{UserName: "u"}, Created: after, Updated: after},
			},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, since)
	require.True(t, iter.NextIssue())

	var bodies []string
	for iter.NextEvent() {
		if e, ok := iter.EventValue().(*CommentEvent); ok {
			bodies = append(bodies, e.Body)
		}
	}
	require.NoError(t, iter.Error())
	assert.Equal(t, []string{"new"}, bodies, "iterator should skip comments updated before since")
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
		TimelineByIssue: map[int64][]*gitea.TimelineComment{
			1: {{ID: 11, Type: "comment", Body: "issue-1-a", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts}},
			2: {{ID: 21, Type: "comment", Body: "issue-2-a", Poster: &gitea.User{UserName: "u"}, Created: ts, Updated: ts}},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	gotByIssue := map[int64][]string{}
	for iter.NextIssue() {
		idx := iter.IssueValue().Index
		for iter.NextEvent() {
			if e, ok := iter.EventValue().(*CommentEvent); ok {
				gotByIssue[idx] = append(gotByIssue[idx], e.Body)
			}
		}
	}
	require.NoError(t, iter.Error())

	assert.Equal(t, []string{"issue-1-a"}, gotByIssue[1])
	assert.Equal(t, []string{"issue-2-a"}, gotByIssue[2])
}
