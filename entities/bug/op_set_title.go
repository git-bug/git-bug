package bug

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/timestamp"

	"github.com/git-bug/git-bug/util/text"
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

	id := op.Id()
	item := &SetTitleTimelineItem{
		combinedId: entity.CombineIds(snapshot.Id(), id),
		Author:     op.Author(),
		UnixTime:   timestamp.Timestamp(op.UnixTime),
		Title:      op.Title,
		Was:        op.Was,
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
	combinedId entity.CombinedId
	Author     identity.Interface
	UnixTime   timestamp.Timestamp
	Title      string
	Was        string
}

func (s SetTitleTimelineItem) CombinedId() entity.CombinedId {
	return s.combinedId
}

// IsAuthored is a sign post method for gqlgen
func (s *SetTitleTimelineItem) IsAuthored() {}

// SetTitle is a convenience function to change a bugs title
func SetTitle(b ReadWrite, author identity.Interface, unixTime int64, title string, metadata map[string]string) (*SetTitleOperation, error) {
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

	op := NewSetTitleOp(author, unixTime, title, was)
	for key, value := range metadata {
		op.SetMetadata(key, value)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)
	return op, nil
}
