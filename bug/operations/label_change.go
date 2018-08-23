package operations

import (
	"fmt"
	"io"
	"io/ioutil"
	"sort"

	"github.com/MichaelMure/git-bug/bug"
)

var _ bug.Operation = LabelChangeOperation{}

// LabelChangeOperation define a Bug operation to add or remove labels
type LabelChangeOperation struct {
	bug.OpBase
	Added   []bug.Label
	Removed []bug.Label
}

// Apply apply the operation
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

func NewLabelChangeOperation(author bug.Person, added, removed []bug.Label) LabelChangeOperation {
	return LabelChangeOperation{
		OpBase:  bug.NewOpBase(bug.LabelChangeOp, author),
		Added:   added,
		Removed: removed,
	}
}

// ChangeLabels is a convenience function to apply the operation
func ChangeLabels(out io.Writer, b bug.Interface, author bug.Person, add, remove []string) error {
	var added, removed []bug.Label

	if out == nil {
		out = ioutil.Discard
	}

	snap := b.Compile()

	for _, str := range add {
		label := bug.Label(str)

		// check for duplicate
		if labelExist(added, label) {
			fmt.Fprintf(out, "label \"%s\" is a duplicate\n", str)
			continue
		}

		// check that the label doesn't already exist
		if labelExist(snap.Labels, label) {
			fmt.Fprintf(out, "label \"%s\" is already set on this bug\n", str)
			continue
		}

		added = append(added, label)
	}

	for _, str := range remove {
		label := bug.Label(str)

		// check for duplicate
		if labelExist(removed, label) {
			fmt.Fprintf(out, "label \"%s\" is a duplicate\n", str)
			continue
		}

		// check that the label actually exist
		if !labelExist(snap.Labels, label) {
			fmt.Fprintf(out, "label \"%s\" doesn't exist on this bug\n", str)
			continue
		}

		removed = append(removed, label)
	}

	if len(added) == 0 && len(removed) == 0 {
		return fmt.Errorf("no label added or removed")
	}

	labelOp := NewLabelChangeOperation(author, added, removed)

	b.Append(labelOp)

	return nil
}

func labelExist(labels []bug.Label, label bug.Label) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}

	return false
}
