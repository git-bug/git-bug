package bug

import (
	"fmt"
	"sort"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"
)

var _ Operation = &LabelChangeOperation{}

// LabelChangeOperation define a Bug operation to add or remove labels
type LabelChangeOperation struct {
	*OpBase
	Added   []Label `json:"added"`
	Removed []Label `json:"removed"`
}

func (op *LabelChangeOperation) base() *OpBase {
	return op.OpBase
}

func (op *LabelChangeOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

// Apply apply the operation
func (op *LabelChangeOperation) Apply(snapshot *Snapshot) {
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

	snapshot.Timeline = append(snapshot.Timeline, op)
}

func (op *LabelChangeOperation) Validate() error {
	if err := opBaseValidate(op, LabelChangeOp); err != nil {
		return err
	}

	for _, l := range op.Added {
		if err := l.Validate(); err != nil {
			return errors.Wrap(err, "added label")
		}
	}

	for _, l := range op.Removed {
		if err := l.Validate(); err != nil {
			return errors.Wrap(err, "removed label")
		}
	}

	if len(op.Added)+len(op.Removed) <= 0 {
		return fmt.Errorf("no label change")
	}

	return nil
}

func NewLabelChangeOperation(author Person, unixTime int64, added, removed []Label) *LabelChangeOperation {
	return &LabelChangeOperation{
		OpBase:  newOpBase(LabelChangeOp, author, unixTime),
		Added:   added,
		Removed: removed,
	}
}

// ChangeLabels is a convenience function to apply the operation
func ChangeLabels(b Interface, author Person, unixTime int64, add, remove []string) ([]LabelChangeResult, error) {
	var added, removed []Label
	var results []LabelChangeResult

	snap := b.Compile()

	for _, str := range add {
		label := Label(str)

		// check for duplicate
		if labelExist(added, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeDuplicateInOp})
			continue
		}

		// check that the label doesn't already exist
		if labelExist(snap.Labels, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeAlreadySet})
			continue
		}

		added = append(added, label)
		results = append(results, LabelChangeResult{Label: label, Status: LabelChangeAdded})
	}

	for _, str := range remove {
		label := Label(str)

		// check for duplicate
		if labelExist(removed, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeDuplicateInOp})
			continue
		}

		// check that the label actually exist
		if !labelExist(snap.Labels, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeDoesntExist})
			continue
		}

		removed = append(removed, label)
		results = append(results, LabelChangeResult{Label: label, Status: LabelChangeRemoved})
	}

	if len(added) == 0 && len(removed) == 0 {
		return results, fmt.Errorf("no label added or removed")
	}

	labelOp := NewLabelChangeOperation(author, unixTime, added, removed)

	if err := labelOp.Validate(); err != nil {
		return nil, err
	}

	b.Append(labelOp)

	return results, nil
}

func labelExist(labels []Label, label Label) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}

	return false
}

type LabelChangeStatus int

const (
	_ LabelChangeStatus = iota
	LabelChangeAdded
	LabelChangeRemoved
	LabelChangeDuplicateInOp
	LabelChangeAlreadySet
	LabelChangeDoesntExist
)

type LabelChangeResult struct {
	Label  Label
	Status LabelChangeStatus
}

func (l LabelChangeResult) String() string {
	switch l.Status {
	case LabelChangeAdded:
		return fmt.Sprintf("label %s added", l.Label)
	case LabelChangeRemoved:
		return fmt.Sprintf("label %s removed", l.Label)
	case LabelChangeDuplicateInOp:
		return fmt.Sprintf("label %s is a duplicate", l.Label)
	case LabelChangeAlreadySet:
		return fmt.Sprintf("label %s was already set", l.Label)
	case LabelChangeDoesntExist:
		return fmt.Sprintf("label %s doesn't exist on this bug", l.Label)
	default:
		panic(fmt.Sprintf("unknown label change status %v", l.Status))
	}
}
