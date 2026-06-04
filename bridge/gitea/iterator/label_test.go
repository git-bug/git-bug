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

// TestLabelIteratorReturnsAPIError verifies that a 500 from the labels
// endpoint surfaces through Error() rather than panicking.
func TestLabelIteratorReturnsAPIError(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:        "owner",
		Project:      "repo",
		LabelErrPage: 1,
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	require.NotPanics(t, func() {
		assert.False(t, iter.NextLabel())
	})
	assert.Error(t, iter.Error())
}

// TestLabelIteratorReturnsLabels verifies the happy path: labels configured
// on the fake flow through LabelValue.
func TestLabelIteratorReturnsLabels(t *testing.T) {
	ts := time.Now()
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "repo",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t",
			Poster: &gitea.User{UserName: "u"}, Created: ts,
		}},
		Labels: []*gitea.Label{
			{ID: 1, Name: "bug"},
			{ID: 2, Name: "enhancement"},
		},
	}
	srv := fa.NewServer(t)

	iter := NewIterator(context.Background(), newTestClient(t, srv.URL), 10, fa.Owner, fa.Project, 5*time.Second, time.Time{})
	require.True(t, iter.NextIssue())

	var names []string
	for iter.NextLabel() {
		names = append(names, iter.LabelValue().Name)
	}
	require.NoError(t, iter.Error())
	assert.Equal(t, []string{"bug", "enhancement"}, names)
}
