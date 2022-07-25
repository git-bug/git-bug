package bug

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/timestamp"

	"github.com/MichaelMure/git-bug/util/text"
)

var _ Operation = &SetTitleOperation{}

// SetTitleOperation will change the title of a bug
type SetTitleOperation struct {
	dag.OpBase
	Title string `json:"title"`
	Was   string `json:"was"`
}

func (op *SetTitleOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *SetTitleOperation) Apply(snapshot *Snapshot) {
	snapshot.Title = op.Title
	snapshot.addActor(op.Author())

	item := &SetTitleTimelineItem{
		id:       op.Id(),
		Author:   op.Author(),
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Title:    op.Title,
		Was:      op.Was,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetTitleOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetTitleOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}

	if !text.SafeOneLine(op.Title) {
		return fmt.Errorf("title has unsafe characters")
	}

	if !text.SafeOneLine(op.Was) {
		return fmt.Errorf("previous title has unsafe characters")
	}

	return nil
}

func NewSetTitleOp(author identity.Interface, unixTime int64, title string, was string) *SetTitleOperation {
	return &SetTitleOperation{
		OpBase: dag.NewOpBase(SetTitleOp, author, unixTime),
		Title:  title,
		Was:    was,
	}
}

type SetTitleTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Title    string
	Was      string
}

func (s SetTitleTimelineItem) Id() entity.Id {
	return s.id
}

// IsAuthored is a sign post method for gqlgen
func (s *SetTitleTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetTitle(b Interface, author identity.Interface, unixTime int64, title string) (*SetTitleOperation, error) {
	var lastTitleOp *SetTitleOperation
	for _, op := range b.Operations() {
		switch op := op.(type) {
		case *SetTitleOperation:
			lastTitleOp = op
		}
	}

	var was string
	if lastTitleOp != nil {
		was = lastTitleOp.Title
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
