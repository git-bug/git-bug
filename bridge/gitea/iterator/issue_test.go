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

// TestIssueIteratorPassesSince documents finding #7: the since value stored in
// conf is never forwarded to ListRepoIssues, so every import fetches all issues
// regardless of the requested cutoff.
//
// After the fix: the `since` query parameter should appear in the API request.
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

// TestIteratorStartsAtPage1 pins Gitea's 1-based pagination contract.
func TestIteratorStartsAtPage1(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	iter.NextIssue()
	require.NoError(t, iter.Error())

	require.Len(t, fa.IssueRequests, 1)
	assert.Equal(t, "1", fa.IssueRequests[0].URL.Query().Get("page"))
}

// TestIteratorNoTrailingIssueCall verifies exact-capacity issue pages stop at
// X-Total-Count instead of probing one more empty page.
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

// TestIteratorRespectsXTotalCount verifies a partial first page, as determined
// by X-Total-Count, does not trigger an extra request.
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

// TestIteratorMultiPageStopsAtTotal verifies the iterator stops exactly on
// the page containing X-Total-Count's final item.
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

// TestIssueIteratorPaginates verifies that fetchIssues walks past the first
// page when X-Total-Count indicates more issues remain.
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

// TestIssueIteratorEmptyRepo verifies that NextIssue on a repo with zero
// issues returns false with no error and without panicking on IssueValue.
func TestIssueIteratorEmptyRepo(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo"}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.NotPanics(t, func() {
		assert.False(t, iter.NextIssue())
	})
	assert.NoError(t, iter.Error())
}

// TestIssueIteratorFiltersToIssues verifies that the issues request asks for
// type=issues, excluding pull requests from the import stream.
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

// TestIssueIteratorReturnsAPIError verifies that a 500 from the issues
// endpoint surfaces through Error() rather than panicking on a nil response.
func TestIssueIteratorReturnsAPIError(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo", IssueErrPage: 1}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	require.NotPanics(t, func() {
		assert.False(t, iter.NextIssue())
	})
	assert.Error(t, iter.Error())
}
