package board

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/text"
)

var _ Operation = &CreateOperation{}

// CreateOperation define the initial creation of a board. The board starts with
// no column and no item: columns are added by operations of their own, typically
// in the same pack.
type CreateOperation struct {
	dag.OpBase
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (op *CreateOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *CreateOperation) Apply(snapshot *Snapshot) {
	// sanity check: will fail when adding a second Create
	if snapshot.id != "" && snapshot.id != entity.UnsetId && snapshot.id != op.Id() {
		return
	}

	snapshot.id = op.Id()

	snapshot.addActor(op.Author())

	snapshot.Title = op.Title
	snapshot.Description = op.Description
	snapshot.CreateTime = op.Time()
}

func (op *CreateOperation) Validate() error {
	if err := op.OpBase.Validate(op, CreateOp); err != nil {
		return err
	}

	if err := validateTitle("title", op.Title); err != nil {
		return err
	}
	if !text.SafeOneLine(op.Description) {
		return fmt.Errorf("description has unsafe characters")
	}

	return nil
}

func NewCreateOp(author identity.Interface, unixTime int64, title, description string) *CreateOperation {
	return &CreateOperation{
		OpBase:      dag.NewOpBase(CreateOp, author, unixTime),
		Title:       title,
		Description: description,
	}
}

// Create is a convenience function to create a board
func Create(author identity.Interface, unixTime int64, title, description string, metadata map[string]string) (*Board, *CreateOperation, error) {
	b := NewBoard()
	op := NewCreateOp(author, unixTime, title, description)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, op, err
	}
	b.Append(op)
	return b, op, nil
}
