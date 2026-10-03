package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
)

// A clone and its original can each have operations applied without the other noticing.
func TestSnapshotClone(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)
	isaac, err := identity.NewIdentity(repo, "Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)

	unix := time.Now().Unix()

	// apply builds a snapshot from scratch, so that two calls with the same
	// operations give independent, equal snapshots
	apply := func(ops ...Operation) *Snapshot {
		snap := &Snapshot{}
		for _, op := range ops {
			op.Apply(snap)
			snap.AppendOperation(op)
		}
		return snap
	}

	create := NewCreateOp(rene, unix, "title", "create", nil)
	base := []Operation{
		create,
		NewAddCommentOp(rene, unix, "comment", nil),
		NewLabelChangeOperation(rene, unix, []common.Label{"first", "second"}, nil),
	}

	// one of each kind of operation that changes the snapshot: the comment edit
	// and the label removal modify existing elements in place, the others append
	more := []Operation{
		NewEditCommentOp(isaac, unix, create.Id(), "create edited", nil),
		NewAddCommentOp(isaac, unix, "another comment", nil),
		NewSetTitleOp(isaac, unix, "edited title", "title"),
		NewSetStatusOp(isaac, unix, common.ClosedStatus),
		NewLabelChangeOperation(isaac, unix, []common.Label{"third"}, []common.Label{"first"}),
	}

	orig := apply(base...)
	clone := orig.Clone()
	require.Equal(t, orig, clone)

	// changing the clone leaves the original untouched
	for _, op := range more {
		op.Apply(clone)
		clone.AppendOperation(op)
	}
	require.Equal(t, apply(base...), orig)
	require.Equal(t, apply(append(base, more...)...), clone)

	// and changing the original leaves the clone untouched
	for _, op := range more {
		op.Apply(orig)
		orig.AppendOperation(op)
	}
	require.Equal(t, apply(append(base, more...)...), clone)
	require.Equal(t, apply(append(base, more...)...), orig)
}
