package board

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/text"
)

var _ Operation = &SetTitleOperation{}

// SetTitleOperation will change the title of a board
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
}

func (op *SetTitleOperation) Validate() error {
	if err := op.OpBase.Validate(op, SetTitleOp); err != nil {
		return err
	}

	if err := validateTitle("title", op.Title); err != nil {
		return err
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

// SetTitle is a convenience function to change a board title
func SetTitle(b Mutable, author identity.Interface, unixTime int64, title string, metadata map[string]string) (*SetTitleOperation, error) {
	was := b.Compile().Title

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
