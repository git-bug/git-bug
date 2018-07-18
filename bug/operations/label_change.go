package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"sort"
)

// LabelChangeOperation will add or remove a set of labels

var _ bug.Operation = LabelChangeOperation{}

type LabelChangeOperation struct {
	bug.OpBase
	Added   []bug.Label
	Removed []bug.Label
}

func NewLabelChangeOperation(author bug.Person, added, removed []bug.Label) LabelChangeOperation {
	return LabelChangeOperation{
		OpBase:  bug.NewOpBase(bug.LabelChangeOp, author),
		Added:   added,
		Removed: removed,
	}
}

func (op LabelChangeOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	// Add in the set
AddLoop:
	for _, added := range op.Added {
		for _, label := range snapshot.Labels {
			if label == added {
				// Already exist
				continue AddLoop
			}
		}

		snapshot.Labels = append(snapshot.Labels, added)
	}

	// Remove in the set
	for _, removed := range op.Removed {
		for i, label := range snapshot.Labels {
			if label == removed {
				snapshot.Labels[i] = snapshot.Labels[len(snapshot.Labels)-1]
				snapshot.Labels = snapshot.Labels[:len(snapshot.Labels)-1]
			}
		}
	}

	// Sort
	sort.Slice(snapshot.Labels, func(i, j int) bool {
		return string(snapshot.Labels[i]) < string(snapshot.Labels[j])
	})

	return snapshot
}
