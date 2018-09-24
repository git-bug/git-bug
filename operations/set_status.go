package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/pkg/errors"
)

// SetStatusOperation will change the status of a bug

var _ bug.Operation = SetStatusOperation{}

type SetStatusOperation struct {
	*bug.OpBase
	Status bug.Status `json:"status"`
}

func (op SetStatusOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Status = op.Status

	return snapshot
}

func (op SetStatusOperation) Validate() error {
	if err := bug.OpBaseValidate(op, bug.SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	return nil
}

func NewSetStatusOp(author bug.Person, status bug.Status) SetStatusOperation {
	return SetStatusOperation{
		OpBase: bug.NewOpBase(bug.SetStatusOp, author),
		Status: status,
	}
}

// Convenience function to apply the operation
func Open(b bug.Interface, author bug.Person) error {
	op := NewSetStatusOp(author, bug.OpenStatus)
	if err := op.Validate(); err != nil {
		return err
	}
	b.Append(op)
	return nil
}

// Convenience function to apply the operation
func Close(b bug.Interface, author bug.Person) error {
	op := NewSetStatusOp(author, bug.ClosedStatus)
	if err := op.Validate(); err != nil {
		return err
	}
	b.Append(op)
	return nil
}
