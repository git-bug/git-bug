package operations

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util"
)

// SetTitleOperation will change the title of a bug

var _ bug.Operation = SetTitleOperation{}

type SetTitleOperation struct {
	bug.OpBase
	Title string
}

func (op SetTitleOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title

	return snapshot
}

func (op SetTitleOperation) Files() []util.Hash {
	return nil
}

func NewSetTitleOp(author bug.Person, title string) SetTitleOperation {
	return SetTitleOperation{
		OpBase: bug.NewOpBase(bug.SetTitleOp, author),
		Title:  title,
	}
}

// Convenience function to apply the operation
func SetTitle(b *bug.Bug, author bug.Person, title string) {
	setTitleOp := NewSetTitleOp(author, title)
	b.Append(setTitleOp)
}
