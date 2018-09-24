package operations

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util/text"
)

// SetTitleOperation will change the title of a bug

var _ bug.Operation = SetTitleOperation{}

type SetTitleOperation struct {
	*bug.OpBase
	Title string `json:"title"`
	Was   string `json:"was"`
}

func (op SetTitleOperation) Apply(snapshot bug.Snapshot) bug.Snapshot {
	snapshot.Title = op.Title

	return snapshot
}

func (op SetTitleOperation) Validate() error {
	if err := bug.OpBaseValidate(op, bug.SetTitleOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}

	if strings.Contains(op.Title, "\n") {
		return fmt.Errorf("title should be a single line")
	}

	if !text.Safe(op.Title) {
		return fmt.Errorf("title should be fully printable")
	}

	if strings.Contains(op.Was, "\n") {
		return fmt.Errorf("previous title should be a single line")
	}

	if !text.Safe(op.Was) {
		return fmt.Errorf("previous title should be fully printable")
	}

	return nil
}

func NewSetTitleOp(author bug.Person, title string, was string) SetTitleOperation {
	return SetTitleOperation{
		OpBase: bug.NewOpBase(bug.SetTitleOp, author),
		Title:  title,
		Was:    was,
	}
}

// Convenience function to apply the operation
func SetTitle(b bug.Interface, author bug.Person, title string) error {
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

	if err := setTitleOp.Validate(); err != nil {
		return err
	}

	b.Append(setTitleOp)
	return nil
}
