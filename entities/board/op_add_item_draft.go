package board

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/text"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &AddItemDraftOperation{}

type AddItemDraftOperation struct {
	dag.OpBase
	ColumnId entity.Id `json:"column"`
	Title    string    `json:"title"`
}

func (op *AddItemDraftOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemDraftOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddItemDraftOp); err != nil {
		return err
	}

	if err := op.ColumnId.Validate(); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}
	if !text.SafeOneLine(op.Title) {
		return fmt.Errorf("title has unsafe characters")
	}

	return nil
}

func (op *AddItemDraftOperation) Apply(snapshot *Snapshot) {
	// Recreate the combined Id to match on
	combinedId := entity.CombineIds(snapshot.Id(), op.ColumnId)

	// search the column
	for _, column := range snapshot.Columns {
		if column.CombinedId == combinedId {
			column.Items = append(column.Items, &Draft{
				combinedId: entity.CombineIds(snapshot.id, op.Id()),
				author:     op.Author(),
				title:      op.Title,
				unixTime:   timestamp.Timestamp(op.UnixTime),
			})

			snapshot.addActor(op.Author())
			return
		}
	}
}

func NewAddItemDraftOp(author identity.Interface, unixTime int64, columnId entity.Id, title string) *AddItemDraftOperation {
	return &AddItemDraftOperation{
		OpBase:   dag.NewOpBase(AddItemDraftOp, author, unixTime),
		ColumnId: columnId,
		Title:    title,
	}
}

// AddItemDraft is a convenience function to add a draft item to a Board
func AddItemDraft(b ReadWrite, author identity.Interface, unixTime int64, columnId entity.Id, title string, metadata map[string]string) (entity.CombinedId, *AddItemDraftOperation, error) {
	op := NewAddItemDraftOp(author, unixTime, columnId, title)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
