package board

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

var _ Operation = &AddColumnOperation{}

// AddColumnOperation adds a column to a board. The column is identified by the id
// of the operation.
type AddColumnOperation struct {
	dag.OpBase
	Name            string `json:"name"`
	Position        string `json:"position"`
	PriorityOrdered bool   `json:"priority_ordered,omitempty"`
}

func (op *AddColumnOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddColumnOperation) UnmarshalJSON(data []byte) error {
	if err := rejectNull(data, "priority_ordered"); err != nil {
		return err
	}
	// the same type without this method
	type plain AddColumnOperation
	return json.Unmarshal(data, (*plain)(op))
}

func (op *AddColumnOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	snapshot.insertColumn(Column{
		Id:              op.Id(),
		CombinedId:      entity.CombineIds(snapshot.Id(), op.Id()),
		Name:            op.Name,
		Position:        op.Position,
		PriorityOrdered: op.PriorityOrdered,
	})
}

func (op *AddColumnOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddColumnOp); err != nil {
		return err
	}

	if err := validateTitle("name", op.Name); err != nil {
		return err
	}
	return validatePosition(op.Position)
}

func NewAddColumnOp(author identity.Interface, unixTime int64, name string, position string, priorityOrdered bool) *AddColumnOperation {
	return &AddColumnOperation{
		OpBase:          dag.NewOpBase(AddColumnOp, author, unixTime),
		Name:            name,
		Position:        position,
		PriorityOrdered: priorityOrdered,
	}
}

// AddColumn is a convenience function to add a column to a board
func AddColumn(b Mutable, author identity.Interface, unixTime int64, name string, position string, priorityOrdered bool, metadata map[string]string) (entity.CombinedId, *AddColumnOperation, error) {
	op := NewAddColumnOp(author, unixTime, name, position, priorityOrdered)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}

var _ Operation = &EditColumnOperation{}

// EditColumnOperation changes a column: its name, its position, whether it is
// priority-ordered, or several of these.
type EditColumnOperation struct {
	dag.OpBase
	Column entity.Id `json:"column"`
	// Name, Position and PriorityOrdered are the new values of the column, nil for
	// the ones that the operation leaves as they are. At least one is set.
	Name            *string `json:"name,omitempty"`
	Position        *string `json:"position,omitempty"`
	PriorityOrdered *bool   `json:"priority_ordered,omitempty"`
}

func (op *EditColumnOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *EditColumnOperation) UnmarshalJSON(data []byte) error {
	if err := rejectNull(data, "name", "position", "priority_ordered"); err != nil {
		return err
	}
	// the same type without this method
	type plain EditColumnOperation
	return json.Unmarshal(data, (*plain)(op))
}

func (op *EditColumnOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	i, ok := snapshot.columnIndex(op.Column)
	if !ok {
		// the column is not present, nothing to change
		return
	}

	// each value is replaced on its own: of two concurrent operations renaming and
	// moving the column, both take effect
	column := snapshot.Columns[i]
	if op.Name != nil {
		column.Name = *op.Name
	}
	if op.Position != nil {
		column.Position = *op.Position
	}
	if op.PriorityOrdered != nil {
		column.PriorityOrdered = *op.PriorityOrdered
	}

	// back at its place in the order, which its position may have changed
	snapshot.Columns = slices.Delete(snapshot.Columns, i, i+1)
	snapshot.insertColumn(column)
}

func (op *EditColumnOperation) Validate() error {
	if err := op.OpBase.Validate(op, EditColumnOp); err != nil {
		return err
	}

	if err := validateTarget("column", op.Column); err != nil {
		return err
	}
	if op.Name == nil && op.Position == nil && op.PriorityOrdered == nil {
		return fmt.Errorf("nothing to change")
	}
	if op.Name != nil {
		if err := validateTitle("name", *op.Name); err != nil {
			return err
		}
	}
	if op.Position != nil {
		if err := validatePosition(*op.Position); err != nil {
			return err
		}
	}

	return nil
}

// NewEditColumnOp makes an operation changing a column. name, position and
// priorityOrdered are the new values, nil for the ones to leave as they are.
func NewEditColumnOp(author identity.Interface, unixTime int64, column entity.Id, name *string, position *string, priorityOrdered *bool) *EditColumnOperation {
	return &EditColumnOperation{
		OpBase:          dag.NewOpBase(EditColumnOp, author, unixTime),
		Column:          column,
		Name:            name,
		Position:        position,
		PriorityOrdered: priorityOrdered,
	}
}

// EditColumn is a convenience function to change a column of a board, see
// NewEditColumnOp
func EditColumn(b Mutable, author identity.Interface, unixTime int64, column entity.Id, name *string, position *string, priorityOrdered *bool, metadata map[string]string) (*EditColumnOperation, error) {
	op := NewEditColumnOp(author, unixTime, column, name, position, priorityOrdered)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

var _ Operation = &RemoveColumnOperation{}

// RemoveColumnOperation removes a column from a board. Its items stay on the
// board, without a column.
type RemoveColumnOperation struct {
	dag.OpBase
	Column entity.Id `json:"column"`
}

func (op *RemoveColumnOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *RemoveColumnOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	i, ok := snapshot.columnIndex(op.Column)
	if !ok {
		// the column is not present, nothing to remove
		return
	}

	snapshot.Columns = slices.Delete(snapshot.Columns, i, i+1)

	// its items keep their position, which doesn't depend on the column
	for i := range snapshot.Items {
		if snapshot.Items[i].Column == op.Column {
			snapshot.Items[i].Column = ""
		}
	}
}

func (op *RemoveColumnOperation) Validate() error {
	if err := op.OpBase.Validate(op, RemoveColumnOp); err != nil {
		return err
	}

	return validateTarget("column", op.Column)
}

func NewRemoveColumnOp(author identity.Interface, unixTime int64, column entity.Id) *RemoveColumnOperation {
	return &RemoveColumnOperation{
		OpBase: dag.NewOpBase(RemoveColumnOp, author, unixTime),
		Column: column,
	}
}

// RemoveColumn is a convenience function to remove a column from a board
func RemoveColumn(b Mutable, author identity.Interface, unixTime int64, column entity.Id, metadata map[string]string) (*RemoveColumnOperation, error) {
	op := NewRemoveColumnOp(author, unixTime, column)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
