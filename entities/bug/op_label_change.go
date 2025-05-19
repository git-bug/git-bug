package bug

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &LabelChangeOperation{}

// LabelChangeOperation define a Bug operation to add or remove labels
type LabelChangeOperation struct {
	dag.OpBase
	Added   []common.Label `json:"added"`
	Removed []common.Label `json:"removed"`
}

func (op *LabelChangeOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

// Apply applies the operation
func (op *LabelChangeOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

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

	id := op.Id()
	item := &LabelChangeTimelineItem{
		// id:         id,
		combinedId: entity.CombineIds(snapshot.Id(), id),
		Author:     op.Author(),
		UnixTime:   timestamp.Timestamp(op.UnixTime),
		Added:      op.Added,
		Removed:    op.Removed,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *LabelChangeOperation) Validate() error {
	if err := op.OpBase.Validate(op, LabelChangeOp); err != nil {
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

func NewLabelChangeOperation(author identity.Interface, unixTime int64, added, removed []common.Label) *LabelChangeOperation {
	return &LabelChangeOperation{
		OpBase:  dag.NewOpBase(LabelChangeOp, author, unixTime),
		Added:   added,
		Removed: removed,
	}
}

type LabelChangeTimelineItem struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	UnixTime   timestamp.Timestamp
	Added      []common.Label
	Removed    []common.Label
}

func (l LabelChangeTimelineItem) CombinedId() entity.CombinedId {
	return l.combinedId
}

// IsAuthored is a sign post method for gqlgen
func (l *LabelChangeTimelineItem) IsAuthored() {}

// ChangeLabels is a convenience function to change labels on a bug
func ChangeLabels(b Interface, author identity.Interface, unixTime int64, add, remove []string, metadata map[string]string) ([]LabelChangeResult, *LabelChangeOperation, error) {
	var added, removed []common.Label
	var results []LabelChangeResult

	snap := b.Compile()

	for _, str := range add {
		label := common.Label(str)

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
		label := common.Label(str)

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
		return results, nil, fmt.Errorf("no label added or removed")
	}

	op := NewLabelChangeOperation(author, unixTime, added, removed)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, nil, err
	}

	b.Append(op)

	return results, op, nil
}

// ForceChangeLabels is a convenience function to apply the operation
// The difference with ChangeLabels is that no checks for deduplication are done. You are entirely
// responsible for what you are doing. In the general case, you want to use ChangeLabels instead.
// The intended use of this function is to allow importers to create legal but unexpected label changes,
// like removing a label with no information of when it was added before.
func ForceChangeLabels(b Interface, author identity.Interface, unixTime int64, add, remove []string, metadata map[string]string) (*LabelChangeOperation, error) {
	added := make([]common.Label, len(add))
	for i, str := range add {
		added[i] = common.Label(str)
	}

	removed := make([]common.Label, len(remove))
	for i, str := range remove {
		removed[i] = common.Label(str)
	}

	op := NewLabelChangeOperation(author, unixTime, added, removed)

	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)

	return op, nil
}

func labelExist(labels []common.Label, label common.Label) bool {
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

func (l LabelChangeStatus) MarshalGQL(w io.Writer) {
	switch l {
	case LabelChangeAdded:
		_, _ = w.Write([]byte(strconv.Quote("ADDED")))
	case LabelChangeRemoved:
		_, _ = w.Write([]byte(strconv.Quote("REMOVED")))
	case LabelChangeDuplicateInOp:
		_, _ = w.Write([]byte(strconv.Quote("DUPLICATE_IN_OP")))
	case LabelChangeAlreadySet:
		_, _ = w.Write([]byte(strconv.Quote("ALREADY_EXIST")))
	case LabelChangeDoesntExist:
		_, _ = w.Write([]byte(strconv.Quote("DOESNT_EXIST")))
	default:
		panic("missing case")
	}
}

func (l *LabelChangeStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	switch str {
	case "ADDED":
		*l = LabelChangeAdded
	case "REMOVED":
		*l = LabelChangeRemoved
	case "DUPLICATE_IN_OP":
		*l = LabelChangeDuplicateInOp
	case "ALREADY_EXIST":
		*l = LabelChangeAlreadySet
	case "DOESNT_EXIST":
		*l = LabelChangeDoesntExist
	default:
		return fmt.Errorf("%s is not a valid LabelChangeStatus", str)
	}
	return nil
}

type LabelChangeResult struct {
	Label  common.Label
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
