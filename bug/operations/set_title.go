package operations

import "github.com/MichaelMure/git-bug/bug"

var _ bug.Operation = SetTitleOperation{}

type SetTitleOperation struct {
	Title string
}

func NewSetTitleOp(title string) SetTitleOperation {
	return SetTitleOperation{
		Title: title,
	}
}

func (op SetTitleOperation) OpType() bug.OperationType {
	return bug.SET_TITLE
}

func (op SetTitleOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title
	return snapshot
}
