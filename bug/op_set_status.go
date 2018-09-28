package bug

import (
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/pkg/errors"
)

// SetStatusOperation will change the status of a bug

var _ Operation = SetStatusOperation{}

type SetStatusOperation struct {
	*OpBase
	Status Status `json:"status"`
}

func (op SetStatusOperation) base() *OpBase {
	return op.OpBase
}

func (op SetStatusOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op SetStatusOperation) Apply(snapshot Snapshot) Snapshot {
	snapshot.Status = op.Status

	return snapshot
}

func (op SetStatusOperation) Validate() error {
	if err := opBaseValidate(op, SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	return nil
}

func NewSetStatusOp(author Person, unixTime int64, status Status) SetStatusOperation {
	return SetStatusOperation{
		OpBase: newOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

// Convenience function to apply the operation
func Open(b Interface, author Person, unixTime int64) error {
	op := NewSetStatusOp(author, unixTime, OpenStatus)
	if err := op.Validate(); err != nil {
		return err
	}
	b.Append(op)
	return nil
}

// Convenience function to apply the operation
func Close(b Interface, author Person, unixTime int64) error {
	op := NewSetStatusOp(author, unixTime, ClosedStatus)
	if err := op.Validate(); err != nil {
		return err
	}
	b.Append(op)
	return nil
}
