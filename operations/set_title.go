package operations

import (
	"github.com/MichaelMure/git-bug/bug"
)

// SetTitleOperation will change the title of a bug

var _ bug.Operation = SetTitleOperation{}

type SetTitleOperation struct {
	bug.OpBase
	Title string `json:"title"`
	Was   string `json:"was"`
}

func (op SetTitleOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title

	return snapshot
}

func NewSetTitleOp(author bug.Person, title string, was string) SetTitleOperation {
	return SetTitleOperation{
		OpBase: bug.NewOpBase(bug.SetTitleOp, author),
		Title:  title,
		Was:    was,
	}
}

// Convenience function to apply the operation
func SetTitle(b bug.Interface, author bug.Person, title string) {
	it := bug.NewOperationIterator(b)

	var lastTitleOp bug.Operation
	for it.Next() {
		op := it.Value()
		if op.OpType() == bug.SetTitleOp {
			lastTitleOp = op
		}
	}

	var was string
	if lastTitleOp != nil {
		was = lastTitleOp.(SetTitleOperation).Title
	} else {
		was = b.FirstOp().(CreateOperation).Title
	}

	setTitleOp := NewSetTitleOp(author, title, was)
	b.Append(setTitleOp)
}
