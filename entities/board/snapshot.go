package board

import (
	"fmt"
	"slices"
	"time"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

type Column struct {
	// id is the identifier of the column within the board context
	Id entity.Id
	// CombinedId is the global identifier of the column
	CombinedId entity.CombinedId
	Name       string
	Items      []Item
}

// Item is the interface that board item (draft, bug ...) need to implement.
type Item interface {
	// CombinedId returns the global identifier of the item.
	CombinedId() entity.CombinedId
}

var _ dag.Snapshot = &Snapshot{}

type Snapshot struct {
	id entity.Id

	CreateTime  time.Time
	Title       string
	Description string
	Columns     []*Column

	// Actors are all the identities that have interacted with the board (add items ...)
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
// the original noticing, and conversely: the slices and the columns that applying
// an operation modifies are copied, the rest is shared.
// Operations, items and identities are shared, as they don't change once created.
func (snap *Snapshot) Clone() *Snapshot {
	clone := *snap
	clone.Actors = slices.Clone(snap.Actors)
	clone.Operations = slices.Clone(snap.Operations)
	clone.Columns = make([]*Column, len(snap.Columns))
	for i, column := range snap.Columns {
		// items are appended in place, see AddItemDraftOperation.Apply
		c := *column
		c.Items = slices.Clone(column.Items)
		clone.Columns[i] = &c
	}
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
			return column, nil
		}
	}

	return nil, fmt.Errorf("column not found")
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

// HasActor return true if the id is a actor
func (snap *Snapshot) HasActor(id entity.Id) bool {
	for _, p := range snap.Actors {
		if p.Id() == id {
			return true
		}
	}
	return false
}

// HasAnyActor return true if one of the ids is a actor
func (snap *Snapshot) HasAnyActor(ids ...entity.Id) bool {
	for _, id := range ids {
		if snap.HasActor(id) {
			return true
		}
	}
	return false
}

// ItemCount returns the number of items (draft, entity) in the board.
func (snap *Snapshot) ItemCount() int {
	var count int
	for _, column := range snap.Columns {
		count += len(column.Items)
	}
	return count
}
