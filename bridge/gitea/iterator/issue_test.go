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

func TestIssueIteratorPassesSince(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)

	since := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, since)
	iter.NextIssue()
	require.NoError(t, iter.Error())

	require.Len(t, fa.IssueRequests, 1)
	capturedSince := fa.IssueRequests[0].URL.Query().Get("since")

	assert.NotEmpty(t, capturedSince,
		"since parameter should be forwarded to ListRepoIssues (finding #7)")

	parsed, err := time.Parse(time.RFC3339, capturedSince)
	require.NoError(t, err, "since value %q should be valid RFC3339", capturedSince)
	assert.Equal(t, since.UTC(), parsed.UTC())
}

func TestIteratorStartsAtPage1(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	iter.NextIssue()
	require.NoError(t, iter.Error())

	require.Len(t, fa.IssueRequests, 1)
	assert.Equal(t, "1", fa.IssueRequests[0].URL.Query().Get("page"))
}

func TestIteratorNoTrailingIssueCall(t *testing.T) {
	const capacity = 2
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "one", Poster: &gitea.User{UserName: "u"}, Created: ts},
			{ID: 2, Index: 2, Title: "two", Poster: &gitea.User{UserName: "u"}, Created: ts},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	for iter.NextIssue() {
	}
	require.NoError(t, iter.Error())
	assert.Len(t, fa.IssueRequests, 1,
		"exactly capacity issues should not trigger a trailing empty issue request")
}

func TestIteratorRespectsXTotalCount(t *testing.T) {
	const capacity = 10
	ts := time.Now()
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	for i := int64(1); i <= 3; i++ {
		fa.Issues = append(fa.Issues, &gitea.Issue{
			ID: i, Index: i, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		})
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	for iter.NextIssue() {
	}
	require.NoError(t, iter.Error())
	assert.Len(t, fa.IssueRequests, 1,
		"X-Total-Count=3 with capacity=10 should stop after the first issue request")
}

func TestIteratorMultiPageStopsAtTotal(t *testing.T) {
	const capacity = 10
	ts := time.Now()
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	for i := int64(1); i <= 25; i++ {
		fa.Issues = append(fa.Issues, &gitea.Issue{
			ID: i, Index: i, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		})
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	var got []*gitea.Issue
	for iter.NextIssue() {
		got = append(got, iter.IssueValue())
	}
	require.NoError(t, iter.Error())
	assert.Len(t, got, 25)
	assert.Len(t, fa.IssueRequests, 3,
		"25 issues at capacity 10 should make exactly 3 issue requests")
}

func TestIssueIteratorPaginates(t *testing.T) {
	const capacity = 2
	ts := time.Now()
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	for i := int64(1); i <= 5; i++ {
		fa.Issues = append(fa.Issues, &gitea.Issue{
			ID: i, Index: i, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		})
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), capacity, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	var got []*gitea.Issue
	for iter.NextIssue() {
		got = append(got, iter.IssueValue())
	}
	require.NoError(t, iter.Error())
	assert.Len(t, got, len(fa.Issues))
}

func TestIssueIteratorEmptyRepo(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.NotPanics(t, func() {
		assert.False(t, iter.NextIssue())
	})
	assert.NoError(t, iter.Error())
}

func TestIssueIteratorFiltersToIssues(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	iter.NextIssue()
	require.NoError(t, iter.Error())

	require.Len(t, fa.IssueRequests, 1)
	assert.Equal(t, "issues", fa.IssueRequests[0].URL.Query().Get("type"),
		"issue iterator should request type=issues to exclude pull requests")
}

// TestIssueIteratorHandlesNetworkError verifies the iterator survives a
// network-level failure (connection refused), where the SDK returns a nil
// *Response alongside the error. fetchIssues must not deref resp.Header.
func TestIssueIteratorHandlesNetworkError(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)
	url := srv.URL
	srv.Close() // force connection refused on next request

	iter := NewIterator(context.Background(), newTestClient(t, url), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	require.NotPanics(t, func() {
		assert.False(t, iter.NextIssue())
	})
	assert.Error(t, iter.Error())
}

func TestIssueIteratorRespectsSince(t *testing.T) {
	since := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	before := since.Add(-24 * time.Hour)
	after := since.Add(24 * time.Hour)

	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "old", Poster: &gitea.User{UserName: "u"}, Created: before, Updated: before},
			{ID: 2, Index: 2, Title: "new", Poster: &gitea.User{UserName: "u"}, Created: after, Updated: after},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, since)

	var titles []string
	for iter.NextIssue() {
		titles = append(titles, iter.IssueValue().Title)
	}
	require.NoError(t, iter.Error())
	assert.Equal(t, []string{"new"}, titles, "iterator should skip issues updated before since")
}

func TestIssueIteratorReturnsAPIError(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo", IssueErrPage: 1}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	require.NotPanics(t, func() {
		assert.False(t, iter.NextIssue())
	})
	assert.Error(t, iter.Error())
}
