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
