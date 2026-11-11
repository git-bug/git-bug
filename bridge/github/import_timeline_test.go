package github

import (
	"context"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestEnsureTimelineItemConvertedToDiscussion(t *testing.T) {
	repo := repotest.NewGoGitRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()

	author, err := backend.Identities().NewRaw("Michael Muré", "no-reply@git-bug.test", "", "", nil, nil)
	require.NoError(t, err)

	b, _, err := backend.Bugs().NewRaw(author, time.Now().Unix(), "converted issue", "", nil, nil)
	require.NoError(t, err)

	gi := &githubImporter{out: make(chan core.ImportResult, 16)}

	item := timelineItem{
		Typename: "ConvertedToDiscussionEvent",
		ConvertedToDiscussionEvent: struct {
			actorEvent
		}{
			actorEvent: actorEvent{
				Id:        githubv4.ID("CVTD_1"),
				CreatedAt: githubv4.DateTime{Time: time.Now()},
				Actor: &actor{
					Typename: "User",
					Login:    "maintainer",
				},
			},
		},
	}

	ctx := context.Background()

	require.NoError(t, gi.ensureTimelineItem(ctx, backend, b, &item))
	require.Equal(t, common.ClosedStatus, b.Snapshot().Status)

	// Replaying the same event must not add a second close operation.
	ops := len(b.Snapshot().Operations)
	require.NoError(t, gi.ensureTimelineItem(ctx, backend, b, &item))
	require.Len(t, b.Snapshot().Operations, ops)
}
