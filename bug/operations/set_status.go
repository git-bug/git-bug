package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util"
)

// SetStatusOperation will change the status of a bug

var _ bug.Operation = SetStatusOperation{}

type SetStatusOperation struct {
	bug.OpBase
	Status bug.Status
}

func (op SetStatusOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Status = op.Status

	return snapshot
}

func (op SetStatusOperation) Files() []util.Hash {
	return nil
}

func NewSetStatusOp(author bug.Person, status bug.Status) SetStatusOperation {
	return SetStatusOperation{
		OpBase: bug.NewOpBase(bug.SetStatusOp, author),
		Status: status,
	}
}

// Convenience function to apply the operation
func Open(b *bug.Bug, author bug.Person) {
	op := NewSetStatusOp(author, bug.OpenStatus)
	b.Append(op)
}

// Convenience function to apply the operation
func Close(b *bug.Bug, author bug.Person) {
	op := NewSetStatusOp(author, bug.ClosedStatus)
	b.Append(op)
}
