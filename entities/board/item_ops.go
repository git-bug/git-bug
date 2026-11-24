package board

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/timestamp"
)

var _ Operation = &AddItemDraftOperation{}

// AddItemDraftOperation adds a draft to a board. The item is identified by the id
// of the operation.
type AddItemDraftOperation struct {
	dag.OpBase
	Title string `json:"title"`
	// Column is the id of the column of the item, empty for none
	Column   entity.Id `json:"column,omitempty"`
	Position string    `json:"position"`
}

func (op *AddItemDraftOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemDraftOperation) UnmarshalJSON(data []byte) error {
	if err := rejectNull(data, "column"); err != nil {
		return err
	}
	// the same type without this method
	type plain AddItemDraftOperation
	return json.Unmarshal(data, (*plain)(op))
}

func (op *AddItemDraftOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	snapshot.insertItem(Item{
		Id:         op.Id(),
		CombinedId: entity.CombineIds(snapshot.Id(), op.Id()),
		Author:     op.Author(),
		UnixTime:   timestamp.Timestamp(op.UnixTime),
		Title:      op.Title,
		Column:     snapshot.placement(op.Column),
		Position:   op.Position,
	})
}

func (op *AddItemDraftOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddItemDraftOp); err != nil {
		return err
	}

	if err := validateTitle("title", op.Title); err != nil {
		return err
	}
	if op.Column != "" {
		if err := validateTarget("column", op.Column); err != nil {
			return err
		}
	}
	return validatePosition(op.Position)
}

// NewAddItemDraftOp makes an operation adding a draft to a board. column is the
// id of its column, empty for none.
func NewAddItemDraftOp(author identity.Interface, unixTime int64, title string, column entity.Id, position string) *AddItemDraftOperation {
	return &AddItemDraftOperation{
		OpBase:   dag.NewOpBase(AddItemDraftOp, author, unixTime),
		Title:    title,
		Column:   column,
		Position: position,
	}
}

// AddItemDraft is a convenience function to add a draft to a board, see
// NewAddItemDraftOp
func AddItemDraft(b Mutable, author identity.Interface, unixTime int64, title string, column entity.Id, position string, metadata map[string]string) (entity.CombinedId, *AddItemDraftOperation, error) {
	op := NewAddItemDraftOp(author, unixTime, title, column, position)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}

var _ Operation = &AddItemEntityOperation{}

// AddItemEntityOperation adds an item referring to an entity, such as a bug, to a
// board. The item is identified by the id of the operation. Only the reference is
// recorded: the entity is not read, and may be absent from the repository.
type AddItemEntityOperation struct {
	dag.OpBase
	Namespace string    `json:"namespace"`
	EntityId  entity.Id `json:"entity"`
	// Column is the id of the column of the item, empty for none
	Column   entity.Id `json:"column,omitempty"`
	Position string    `json:"position"`
}

func (op *AddItemEntityOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *AddItemEntityOperation) UnmarshalJSON(data []byte) error {
	if err := rejectNull(data, "column"); err != nil {
		return err
	}
	// the same type without this method
	type plain AddItemEntityOperation
	return json.Unmarshal(data, (*plain)(op))
}

// Reference returns the entity that the operation adds.
func (op *AddItemEntityOperation) Reference() Reference {
	return Reference{Namespace: op.Namespace, Id: op.EntityId}
}

func (op *AddItemEntityOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	if _, held := snapshot.holder(op.EntityId); held {
		// A board holds an entity once. What is done to the entity next applies to
		// the item that holds it, see Snapshot.itemIndex.
		return
	}

	snapshot.insertItem(Item{
		Id:         op.Id(),
		CombinedId: entity.CombineIds(snapshot.Id(), op.Id()),
		Author:     op.Author(),
		UnixTime:   timestamp.Timestamp(op.UnixTime),
		Entity:     op.Reference(),
		Column:     snapshot.placement(op.Column),
		Position:   op.Position,
	})
}

func (op *AddItemEntityOperation) Validate() error {
	if err := op.OpBase.Validate(op, AddItemEntityOp); err != nil {
		return err
	}

	if err := op.Reference().Validate(); err != nil {
		return err
	}
	if op.Column != "" {
		if err := validateTarget("column", op.Column); err != nil {
			return err
		}
	}
	return validatePosition(op.Position)
}

// NewAddItemEntityOp makes an operation adding an entity to a board. column is the
// id of the column of the item, empty for none.
func NewAddItemEntityOp(author identity.Interface, unixTime int64, ref Reference, column entity.Id, position string) *AddItemEntityOperation {
	return &AddItemEntityOperation{
		OpBase:    dag.NewOpBase(AddItemEntityOp, author, unixTime),
		Namespace: ref.Namespace,
		EntityId:  ref.Id,
		Column:    column,
		Position:  position,
	}
}

// AddItemEntity is a convenience function to add an entity to a board, see
// NewAddItemEntityOp
func AddItemEntity(b Mutable, author identity.Interface, unixTime int64, ref Reference, column entity.Id, position string, metadata map[string]string) (entity.CombinedId, *AddItemEntityOperation, error) {
	op := NewAddItemEntityOp(author, unixTime, ref, column, position)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return entity.UnsetCombinedId, nil, err
	}
	b.Append(op)
	return entity.CombineIds(b.Id(), op.Id()), op, nil
}

var _ Operation = &MoveItemOperation{}

// MoveItemOperation moves an item: to another column, to another place in the
// order of the board, or both.
type MoveItemOperation struct {
	dag.OpBase
	// Item designates the item: by the id of the entity it refers to, or by its
	// own id for a draft.
	Item entity.Id
	// Column is where the item goes: nil to leave it in its column, otherwise the
	// id of a column, or the empty id for no column.
	Column   *entity.Id
	Position string
}

// moveItemJSON is the JSON of a MoveItemOperation. Its column has three forms:
// absent for an item left in its column, null for no column, or the id of a
// column.
type moveItemJSON struct {
	dag.OpBase
	Item     entity.Id       `json:"item"`
	Column   json.RawMessage `json:"column,omitempty"`
	Position string          `json:"position"`
}

func (op *MoveItemOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *MoveItemOperation) MarshalJSON() ([]byte, error) {
	aux := moveItemJSON{OpBase: op.OpBase, Item: op.Item, Position: op.Position}
	switch {
	case op.Column == nil:
		// left out
	case *op.Column == "":
		aux.Column = json.RawMessage("null")
	default:
		column, err := json.Marshal(*op.Column)
		if err != nil {
			return nil, err
		}
		aux.Column = column
	}
	return json.Marshal(aux)
}

func (op *MoveItemOperation) UnmarshalJSON(data []byte) error {
	var aux moveItemJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	op.OpBase, op.Item, op.Position = aux.OpBase, aux.Item, aux.Position

	switch {
	case aux.Column == nil:
		op.Column = nil
	case string(aux.Column) == "null":
		op.Column = new(entity.Id)
	default:
		var column entity.Id
		if err := json.Unmarshal(aux.Column, &column); err != nil {
			return fmt.Errorf("column: %w", err)
		}
		if column == "" {
			// no column is null, see moveItemJSON
			return errors.New("column: empty id")
		}
		op.Column = &column
	}
	return nil
}

func (op *MoveItemOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	i, ok := snapshot.itemIndex(op.Item)
	if !ok {
		// the item is not present, nothing to move
		return
	}

	// The position and the column are replaced each on its own: a move within a
	// column doesn't undo a concurrent move to another column.
	item := snapshot.Items[i]
	item.Position = op.Position
	if op.Column != nil {
		item.Column = snapshot.placement(*op.Column)
	}

	// back at its place in the order
	snapshot.Items = slices.Delete(snapshot.Items, i, i+1)
	snapshot.insertItem(item)
}

func (op *MoveItemOperation) Validate() error {
	if err := op.OpBase.Validate(op, MoveItemOp); err != nil {
		return err
	}

	if err := validateTarget("item", op.Item); err != nil {
		return err
	}
	if op.Column != nil && *op.Column != "" {
		if err := validateTarget("column", *op.Column); err != nil {
			return err
		}
	}
	return validatePosition(op.Position)
}

// NewMoveItemOp makes an operation moving an item to a position. item is the id
// of the entity that the item refers to, or its own id for a draft. column is
// where the item goes: nil to leave it in its column, otherwise the id of a
// column, or the empty id for no column.
func NewMoveItemOp(author identity.Interface, unixTime int64, item entity.Id, column *entity.Id, position string) *MoveItemOperation {
	return &MoveItemOperation{
		OpBase:   dag.NewOpBase(MoveItemOp, author, unixTime),
		Item:     item,
		Column:   column,
		Position: position,
	}
}

// MoveItem is a convenience function to move an item of a board, see
// NewMoveItemOp
func MoveItem(b Mutable, author identity.Interface, unixTime int64, item entity.Id, column *entity.Id, position string, metadata map[string]string) (*MoveItemOperation, error) {
	op := NewMoveItemOp(author, unixTime, item, column, position)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

var _ Operation = &RemoveItemOperation{}

// RemoveItemOperation removes an item from a board. The entity that the item
// refers to, if any, is left as it is.
type RemoveItemOperation struct {
	dag.OpBase
	// Item designates the item: by the id of the entity it refers to, or by its
	// own id for a draft.
	Item entity.Id `json:"item"`
}

func (op *RemoveItemOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *RemoveItemOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	i, ok := snapshot.itemIndex(op.Item)
	if !ok {
		// the item is not present, nothing to remove
		return
	}

	snapshot.Items = slices.Delete(snapshot.Items, i, i+1)
}

func (op *RemoveItemOperation) Validate() error {
	if err := op.OpBase.Validate(op, RemoveItemOp); err != nil {
		return err
	}

	return validateTarget("item", op.Item)
}

// NewRemoveItemOp makes an operation removing an item. item is the id of the
// entity that the item refers to, or its own id for a draft.
func NewRemoveItemOp(author identity.Interface, unixTime int64, item entity.Id) *RemoveItemOperation {
	return &RemoveItemOperation{
		OpBase: dag.NewOpBase(RemoveItemOp, author, unixTime),
		Item:   item,
	}
}

// RemoveItem is a convenience function to remove an item from a board
func RemoveItem(b Mutable, author identity.Interface, unixTime int64, item entity.Id, metadata map[string]string) (*RemoveItemOperation, error) {
	op := NewRemoveItemOp(author, unixTime, item)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

var _ Operation = &EditDraftOperation{}

// EditDraftOperation changes the title of a draft.
type EditDraftOperation struct {
	dag.OpBase
	Item  entity.Id `json:"item"`
	Title string    `json:"title"`
}

func (op *EditDraftOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

func (op *EditDraftOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	i, ok := snapshot.itemIndex(op.Item)
	if !ok || !snapshot.Items[i].IsDraft() {
		// the item is not present, or was converted into an entity: nothing to edit
		return
	}

	snapshot.Items[i].Title = op.Title
}

func (op *EditDraftOperation) Validate() error {
	if err := op.OpBase.Validate(op, EditDraftOp); err != nil {
		return err
	}

	if err := validateTarget("item", op.Item); err != nil {
		return err
	}
	return validateTitle("title", op.Title)
}

func NewEditDraftOp(author identity.Interface, unixTime int64, item entity.Id, title string) *EditDraftOperation {
	return &EditDraftOperation{
		OpBase: dag.NewOpBase(EditDraftOp, author, unixTime),
		Item:   item,
		Title:  title,
	}
}

// EditDraft is a convenience function to change the title of a draft of a board
func EditDraft(b Mutable, author identity.Interface, unixTime int64, item entity.Id, title string, metadata map[string]string) (*EditDraftOperation, error) {
	op := NewEditDraftOp(author, unixTime, item, title)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

var _ Operation = &ConvertDraftOperation{}

// ConvertDraftOperation turns a draft into a reference to an entity: the bug made
// out of the draft. The item keeps its id, its column and its position.
//
// The entity is created on its own, by its own operations. The two are fetched
// separately: a board may be read before the entity that an item refers to is
// there.
type ConvertDraftOperation struct {
	dag.OpBase
	Item      entity.Id `json:"item"`
	Namespace string    `json:"namespace"`
	EntityId  entity.Id `json:"entity"`
}

func (op *ConvertDraftOperation) Id() entity.Id {
	return dag.IdOperation(op, &op.OpBase)
}

// Reference returns the entity that the draft becomes.
func (op *ConvertDraftOperation) Reference() Reference {
	return Reference{Namespace: op.Namespace, Id: op.EntityId}
}

func (op *ConvertDraftOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author())

	i, ok := snapshot.itemIndex(op.Item)
	if !ok || !snapshot.Items[i].IsDraft() {
		// the item is not present, or was converted already: nothing to convert
		return
	}
	if _, held := snapshot.holder(op.EntityId); held {
		// a board holds an entity once: the draft stays one
		return
	}

	snapshot.Items[i].Title = ""
	snapshot.Items[i].Entity = op.Reference()
}

func (op *ConvertDraftOperation) Validate() error {
	if err := op.OpBase.Validate(op, ConvertDraftOp); err != nil {
		return err
	}

	if err := validateTarget("item", op.Item); err != nil {
		return err
	}
	return op.Reference().Validate()
}

func NewConvertDraftOp(author identity.Interface, unixTime int64, item entity.Id, ref Reference) *ConvertDraftOperation {
	return &ConvertDraftOperation{
		OpBase:    dag.NewOpBase(ConvertDraftOp, author, unixTime),
		Item:      item,
		Namespace: ref.Namespace,
		EntityId:  ref.Id,
	}
}

// ConvertDraft is a convenience function to turn a draft of a board into a
// reference to an entity
func ConvertDraft(b Mutable, author identity.Interface, unixTime int64, item entity.Id, ref Reference, metadata map[string]string) (*ConvertDraftOperation, error) {
	op := NewConvertDraftOp(author, unixTime, item, ref)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
