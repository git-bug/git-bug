package bug

import (
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &SetAssigneeOperation{}

// SetAssigneeOperation will change the assignee of a bug
type SetAssigneeOperation struct {
	dag.OpBase
	Assignee entity.Id `json:"assignee"` // empty Id means unassigned
	// Not serialized. Resolved during unmarshal.
	assignee identity.Interface
}

func (op *SetAssigneeOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetAssigneeOperation) Apply(snapshot *Snapshot) {
	snapshot.Assignee = op.assignee
	snapshot.addActor(op.Author())

	item := &SetAssigneeTimelineItem{
		combinedId: entity.CombineIds(snapshot.Id(), op.Id()),
		Author:     op.Author(),
		UnixTime:   timestamp.Timestamp(op.UnixTime),
		Assignee:   op.assignee,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetAssigneeOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetAssigneeOp); err != nil {
		return err
	}
	if op.Assignee != "" {
		if err := op.Assignee.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func NewSetAssigneeOp(author identity.Interface, unixTime int64, assigneeId entity.Id, assignee identity.Interface) *SetAssigneeOperation {
	return &SetAssigneeOperation{
		OpBase:   dag.NewOpBase(SetAssigneeOp, author, unixTime),
		Assignee: assigneeId,
		assignee: assignee,
	}
}

type SetAssigneeTimelineItem struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	UnixTime   timestamp.Timestamp
	Assignee   identity.Interface
}

func (s SetAssigneeTimelineItem) CombinedId() entity.CombinedId {
	return s.combinedId
}

// IsAuthored is a sign post method for gqlgen
func (s *SetAssigneeTimelineItem) IsAuthored() {}

// SetAssignee is a convenience function to change a bug's assignee
func SetAssignee(b Interface, author identity.Interface, unixTime int64, assigneeId entity.Id, assignee identity.Interface, metadata map[string]string) (*SetAssigneeOperation, error) {
	op := NewSetAssigneeOp(author, unixTime, assigneeId, assignee)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Unassign is a convenience function to remove the assignee from a bug
func Unassign(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*SetAssigneeOperation, error) {
	return SetAssignee(b, author, unixTime, "", nil, metadata)
}
