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

// TestStickyError verifies that once an API error is recorded, subsequent
// NextX calls short-circuit (return false) without further API calls.
func TestStickyError(t *testing.T) {
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "repo", IssueErrPage: 1}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	assert.False(t, iter.NextIssue())
	require.Error(t, iter.Error())

	before := len(fa.IssueRequests)
	assert.False(t, iter.NextIssue())
	assert.False(t, iter.NextIssue())
	assert.Equal(t, before, len(fa.IssueRequests),
		"no further API calls should be made after a sticky error")
}

// TestContextCancellationShortCircuits verifies that a canceled context
// causes NextIssue to return false without setting Error (cancellation is
// not a failure to surface).
func TestContextCancellationShortCircuits(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	iter := NewIterator(ctx, newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})

	require.True(t, iter.NextIssue())
	cancel()

	assert.False(t, iter.NextIssue())
	assert.NoError(t, iter.Error(),
		"context cancellation should short-circuit without surfacing as an iterator error")
}
