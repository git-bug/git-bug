package bug

import (
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &SetStatusOperation{}

// SetStatusOperation will change the status of a bug
type SetStatusOperation struct {
	dag.OpBase
	Status common.Status `json:"status"`
}

func (op *SetStatusOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetStatusOperation) Apply(snapshot *Snapshot) {
	snapshot.Status = op.Status
	snapshot.addActor(op.Author())

	id := op.Id()
	item := &SetStatusTimelineItem{
		// id:         id,
		combinedId: entity.CombineIds(snapshot.Id(), id),
		Author:     op.Author(),
		UnixTime:   timestamp.Timestamp(op.UnixTime),
		Status:     op.Status,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetStatusOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	return nil
}

func NewSetStatusOp(author identity.Interface, unixTime int64, status common.Status) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase: dag.NewOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

type SetStatusTimelineItem struct {
	combinedId entity.CombinedId
	Author     identity.Interface
	UnixTime   timestamp.Timestamp
	Status     common.Status
}

func (s SetStatusTimelineItem) CombinedId() entity.CombinedId {
	return s.combinedId
}

// IsAuthored is a sign post method for gqlgen
func (s *SetStatusTimelineItem) IsAuthored() {}

// Open is a convenience function to change a bugs state to Open
func Open(b ReadWrite, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, common.OpenStatus)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Close is a convenience function to change a bugs state to Close
func Close(b ReadWrite, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, common.ClosedStatus)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
