package bug

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &SetTitleOperation{}

// SetTitleOperation will change the title of a bug
type SetTitleOperation struct {
	*OpBase
	Title string `json:"title"`
	Was   string `json:"was"`
}

func (op *SetTitleOperation) base() *OpBase {
	return op.OpBase
}

func (op *SetTitleOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *SetTitleOperation) Apply(snapshot *Snapshot) {
	snapshot.Title = op.Title
	snapshot.Timeline = append(snapshot.Timeline, op)
}

func (op *SetTitleOperation) Validate() error {
	if err := opBaseValidate(op, SetTitleOp); err != nil {
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

func NewSetTitleOp(author Person, unixTime int64, title string, was string) *SetTitleOperation {
	return &SetTitleOperation{
		OpBase: newOpBase(SetTitleOp, author, unixTime),
		Title:  title,
		Was:    was,
	}
}

// Convenience function to apply the operation
func SetTitle(b Interface, author Person, unixTime int64, title string) error {
	it := NewOperationIterator(b)

	var lastTitleOp Operation
	for it.Next() {
		op := it.Value()
		if op.base().OperationType == SetTitleOp {
			lastTitleOp = op
		}
	}

	var was string
	if lastTitleOp != nil {
		was = lastTitleOp.(*SetTitleOperation).Title
	} else {
		was = b.FirstOp().(*CreateOperation).Title
	}

	setTitleOp := NewSetTitleOp(author, unixTime, title, was)

	if err := setTitleOp.Validate(); err != nil {
		return err
	}

	b.Append(setTitleOp)
	return nil
}
