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
	OpBase
	Title string `json:"title"`
	Was   string `json:"was"`
}

func (op *SetTitleOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetTitleOperation) Hash() (git.Hash, error) {
	return hashOperation(op)
}

func (op *SetTitleOperation) Apply(snapshot *Snapshot) {
	snapshot.Title = op.Title

	hash, err := op.Hash()
	if err != nil {
		// Should never error unless a programming error happened
		// (covered in OpBase.Validate())
		panic(err)
	}

	item := &SetTitleTimelineItem{
		hash:     hash,
		Author:   op.Author,
		UnixTime: Timestamp(op.UnixTime),
		Title:    op.Title,
		Was:      op.Was,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
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

type SetTitleTimelineItem struct {
	hash     git.Hash
	Author   Person
	UnixTime Timestamp
	Title    string
	Was      string
}

func (s SetTitleTimelineItem) Hash() git.Hash {
	return s.hash
}

// Convenience function to apply the operation
func SetTitle(b Interface, author Person, unixTime int64, title string) (*SetTitleOperation, error) {
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
		return nil, err
	}

	b.Append(setTitleOp)
	return setTitleOp, nil
}
