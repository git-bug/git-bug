package operations

import "github.com/MichaelMure/git-bug/bug"

var _ bug.Operation = SetTitleOperation{}

type SetTitleOperation struct {
	bug.OpBase
	Title string `json:"t"`
}

func NewSetTitleOp(title string) SetTitleOperation {
	return SetTitleOperation{
		OpBase: bug.OpBase{OperationType: bug.SET_TITLE},
		Title:  title,
	}
}

func (op SetTitleOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title
	return snapshot
}
