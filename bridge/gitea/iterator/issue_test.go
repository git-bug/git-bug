package iterator

import (
	"context"
	"testing"
	"time"

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
