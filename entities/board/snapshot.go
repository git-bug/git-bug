package board

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

var _ dag.Snapshot = &Snapshot{}

// Snapshot is a compiled form of the Board data structure
type Snapshot struct {
	id entity.Id

	Title       string
	Description string
	CreateTime  time.Time

	// Columns are the columns of the board, in their order: by position, then by
	// id.
	Columns []Column
	// Items are the items of the board, in the order of the board: by position,
	// then by id, whatever their column. The items of a column are in that same
	// order, see ItemsOf.
	Items []Item

	// Actors are the identities that made an operation on the board
	Actors []identity.Interface

	Operations []dag.Operation
}

// Id returns the Board identifier
func (snap *Snapshot) Id() entity.Id {
	if snap.id == "" {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no id")
	}
	return snap.id
}

func (snap *Snapshot) AllOperations() []dag.Operation {
	return snap.Operations
}

func (snap *Snapshot) AppendOperation(op dag.Operation) {
	snap.Operations = append(snap.Operations, op)
}

// Clone returns a copy of the snapshot that operations can be applied to without
// the original noticing, and conversely. Operations and identities are shared, as
// they don't change once created.
func (snap *Snapshot) Clone() *Snapshot {
	clone := *snap
	clone.Columns = slices.Clone(snap.Columns)
	clone.Items = slices.Clone(snap.Items)
	clone.Actors = slices.Clone(snap.Actors)
	clone.Operations = slices.Clone(snap.Operations)
	return &clone
}

// EditTime returns the last time the board was modified
func (snap *Snapshot) EditTime() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}

// SearchColumn will search for a column matching the given id
func (snap *Snapshot) SearchColumn(id entity.CombinedId) (*Column, error) {
	for _, column := range snap.Columns {
		if column.CombinedId == id {
			return &column, nil
		}
	}

	return nil, fmt.Errorf("column not found")
}

// SearchItem will search for an item matching the given id
func (snap *Snapshot) SearchItem(id entity.CombinedId) (*Item, error) {
	for _, item := range snap.Items {
		if item.CombinedId == id {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("item not found")
}

// ItemsOf returns the items of a column, in order. An empty id gives the items
// without a column.
func (snap *Snapshot) ItemsOf(column entity.Id) []Item {
	var result []Item
	for _, item := range snap.Items {
		if item.Column == column {
			result = append(result, item)
		}
	}
	return result
}

// append the operation author to the actors list
func (snap *Snapshot) addActor(actor identity.Interface) {
	for _, a := range snap.Actors {
		if actor.Id() == a.Id() {
			return
		}
	}

	snap.Actors = append(snap.Actors, actor)
}

// compareOrder orders columns, and items, by position then by id.
func compareOrder(positionA string, idA entity.Id, positionB string, idB entity.Id) int {
	return cmp.Or(
		strings.Compare(positionA, positionB),
		strings.Compare(idA.String(), idB.String()),
	)
}

// columnIndex returns the index of the column with this id, false if it is not
// present.
func (snap *Snapshot) columnIndex(id entity.Id) (int, bool) {
	i := slices.IndexFunc(snap.Columns, func(column Column) bool { return column.Id == id })
	return i, i >= 0
}

// hasColumn tells whether the column with this id is present.
func (snap *Snapshot) hasColumn(id entity.Id) bool {
	_, ok := snap.columnIndex(id)
	return ok
}

// insertColumn adds a column at its place in the order.
func (snap *Snapshot) insertColumn(column Column) {
	i, _ := slices.BinarySearchFunc(snap.Columns, column, func(a, b Column) int {
		return compareOrder(a.Position, a.Id, b.Position, b.Id)
	})
	snap.Columns = slices.Insert(snap.Columns, i, column)
}

// itemIndex returns the index of the item that an operation designates by this
// id: the item with this id, otherwise the item referring to the entity with this
// id. False if no such item is present.
//
// Operations designate an item referring to an entity by the id of the entity, so
// that two people adding the same bug at the same time name the same item. The id
// of the item designates it as well: an operation made on a draft still applies
// once the draft is converted.
func (snap *Snapshot) itemIndex(id entity.Id) (int, bool) {
	i := slices.IndexFunc(snap.Items, func(item Item) bool { return item.Id == id })
	if i >= 0 {
		return i, true
	}
	return snap.holder(id)
}

// holder returns the index of the item referring to the entity with this id,
// false if the board doesn't hold it. A board holds an entity once, whatever the
// namespace of the reference.
func (snap *Snapshot) holder(id entity.Id) (int, bool) {
	i := slices.IndexFunc(snap.Items, func(item Item) bool {
		return !item.IsDraft() && item.Entity.Id == id
	})
	return i, i >= 0
}

// insertItem adds an item at its place in the order.
func (snap *Snapshot) insertItem(item Item) {
	i, _ := slices.BinarySearchFunc(snap.Items, item, func(a, b Item) int {
		return compareOrder(a.Position, a.Id, b.Position, b.Id)
	})
	snap.Items = slices.Insert(snap.Items, i, item)
}

// placement returns the column that an operation gives to an item: the one it
// names if it is present, otherwise none.
func (snap *Snapshot) placement(column entity.Id) entity.Id {
	if column != "" && snap.hasColumn(column) {
		return column
	}
	return ""
}
