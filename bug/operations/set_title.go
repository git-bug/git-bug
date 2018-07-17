package operations

import "github.com/MichaelMure/git-bug/bug"

var _ bug.Operation = SetTitleOperation{}

type SetTitleOperation struct {
	bug.OpBase
	Title string
}

func NewSetTitleOp(title string) SetTitleOperation {
	return SetTitleOperation{
		OpBase: bug.OpBase{OperationType: bug.SetTitleOp},
		Title:  title,
	}
}

func (op SetTitleOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title
	return snapshot
}
