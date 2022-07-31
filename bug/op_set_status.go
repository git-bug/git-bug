package bug

import (
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

var _ Operation = &SetStatusOperation{}

// SetStatusOperation will change the status of a bug
type SetStatusOperation struct {
	dag.OpBase
	Status Status `json:"status"`
}

func (op *SetStatusOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetStatusOperation) Apply(snapshot *Snapshot) {
	snapshot.Status = op.Status
	snapshot.addActor(op.Author())

	item := &SetStatusTimelineItem{
		id:       op.Id(),
		Author:   op.Author(),
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Status:   op.Status,
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

func NewSetStatusOp(author identity.Interface, unixTime int64, status Status) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase: dag.NewOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

type SetStatusTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Status   Status
}

func (s SetStatusTimelineItem) Id() entity.Id {
	return s.id
}

// IsAuthored is a sign post method for gqlgen
func (s SetStatusTimelineItem) IsAuthored() {}

// Open is a convenience function to change a bugs state to Open
func Open(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, OpenStatus)
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
func Close(b Interface, author identity.Interface, unixTime int64, metadata map[string]string) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, ClosedStatus)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
