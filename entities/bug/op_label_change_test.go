package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/config"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
	"github.com/git-bug/git-bug/repository"
)

func TestLabelChangeSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, []common.Label{"added"}, []common.Label{"removed"}), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, []common.Label{"added"}, nil), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, nil, []common.Label{"removed"}), nil
	})
}

// A label change is judged by the allowed labels of its era, see config.md §9.3
// in the spec.
func TestLabelChangeAllowed(t *testing.T) {
	repo := repository.NewMockRepo()
	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	require.NoError(t, rene.Commit(t.Context(), repo))
	isaac, err := identity.NewIdentity(repo, "Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)
	require.NoError(t, isaac.Commit(t.Context(), repo))
	unix := time.Now().Unix()

	commit := func(b *Bug, op Operation) {
		t.Helper()
		b.Append(op)
		require.NoError(t, b.Commit(t.Context(), repo))
	}

	// labels added before the restriction
	b, _, err := Create(rene, unix, "title", "message", nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context(), repo))
	commit(b, NewLabelChangeOperation(rene, unix, []common.Label{"kept", "removed"}, nil))

	// the restriction applies to the bug operations made after it
	c, _, err := config.Create(repo, rene, unix)
	require.NoError(t, err)
	_, err = config.LabelsAllow(c, repo, rene, unix, "bug")
	require.NoError(t, err)
	cfg := c.Compile()

	// in part
	partial := NewLabelChangeOperation(isaac, unix, []common.Label{"bug", "wontfix"}, []common.Label{"removed"})
	commit(b, partial)
	// as a whole
	refused := NewLabelChangeOperation(isaac, unix, []common.Label{"wontfix"}, nil)
	commit(b, refused)

	t.Run("judged by its era", func(t *testing.T) {
		snap := b.CompileWith(cfg)
		require.Same(t, cfg, snap.Config())

		// granted before the restriction, a label stays
		require.Equal(t, []common.Label{"bug", "kept"}, snap.Labels)

		// what is refused is left out, timeline included
		require.Len(t, snap.Timeline, 3)
		item := snap.Timeline[2].(*LabelChangeTimelineItem)
		require.Equal(t, entity.CombineIds(b.Id(), partial.Id()), item.CombinedId())
		require.Equal(t, []common.Label{"bug"}, item.Added)
		require.Equal(t, []common.Label{"removed"}, item.Removed)
		require.Len(t, snap.Actors, 2)

		// the operations stay, as valid as any
		require.Len(t, snap.Operations, 4)
		require.Equal(t, refused.Id(), snap.Operations[3].Id())
	})

	t.Run("not committed yet", func(t *testing.T) {
		// judged by the latest config
		read, err := Read(t.Context(), repo, b.Id())
		require.NoError(t, err)
		read.Append(NewLabelChangeOperation(isaac, unix, []common.Label{"staged"}, nil))

		snap := read.CompileWith(cfg)
		require.Equal(t, []common.Label{"bug", "kept"}, snap.Labels)
		require.Len(t, snap.Timeline, 3)
	})

	t.Run("without config", func(t *testing.T) {
		snap := b.Compile()
		require.Equal(t, []common.Label{"bug", "kept", "wontfix"}, snap.Labels)
		require.Len(t, snap.Timeline, 4)
	})
}
