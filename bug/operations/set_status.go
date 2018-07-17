package operations

import (
	"github.com/MichaelMure/git-bug/bug"
)

// SetStatusOperation will change the status of a bug

var _ bug.Operation = SetStatusOperation{}

type SetStatusOperation struct {
	bug.OpBase
	Status bug.Status
}

func NewSetStatusOp(author bug.Person, status bug.Status) SetStatusOperation {
	return SetStatusOperation{
		OpBase: bug.NewOpBase(bug.SetStatusOp, author),
		Status: status,
	}
}

func (op SetStatusOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Status = op.Status

	return snapshot
}
