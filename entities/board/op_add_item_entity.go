package board

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// ItemEntityType indicate the type of entity board item
type ItemEntityType string

const (
	EntityTypeBug ItemEntityType = "bug"
)

var _ Operation = &AddItemEntityOperation{}

// AddItemEntityOperation adds an item referring to another entity (a bug ...) to a
// column. Only the entity's id is recorded: the entity is not loaded with the board.
type AddItemEntityOperation struct {
	dag.OpBase
	ColumnId   entity.Id      `json:"column"`
	EntityType ItemEntityType `json:"entity_type"`
	EntityId   entity.Id      `json:"entity_id"`
}

func (op *AddItemEntityOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemEntityOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddItemEntityOp); err != nil {
		return err
	}

	if err := op.ColumnId.Validate(); err != nil {
		return err
	}

	switch op.EntityType {
	case EntityTypeBug:
	default:
		return fmt.Errorf("unknown entity type")
	}

	if err := op.EntityId.Validate(); err != nil {
		return err
	}

	return nil
}

func (op *AddItemEntityOperation) Apply(snapshot *Snapshot) {
	// Recreate the combined Id to match on
	combinedId := entity.CombineIds(snapshot.Id(), op.ColumnId)

	// search the column
	for _, column := range snapshot.Columns {
		if column.CombinedId == combinedId {
			switch op.EntityType {
			case EntityTypeBug:
				column.Items = append(column.Items, &BugItem{
					combinedId: entity.CombineIds(snapshot.Id(), op.Id()),
					BugId:      op.EntityId,
				})
			}
			snapshot.addActor(op.Author())
			return
		}
	}
}

func NewAddItemEntityOp(author identity.Interface, unixTime int64, columnId entity.Id, entityType ItemEntityType, entityId entity.Id) *AddItemEntityOperation {
	return &AddItemEntityOperation{
		OpBase:     dag.NewOpBase(AddItemEntityOp, author, unixTime),
		ColumnId:   columnId,
		EntityType: entityType,
		EntityId:   entityId,
	}
}

// AddItemEntity is a convenience function to add an entity item to a Board
func AddItemEntity(b Mutable, author identity.Interface, unixTime int64, columnId entity.Id, entityType ItemEntityType, entityId entity.Id, metadata map[string]string) (entity.CombinedId, *AddItemEntityOperation, error) {
	op := NewAddItemEntityOp(author, unixTime, columnId, entityType, entityId)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}
