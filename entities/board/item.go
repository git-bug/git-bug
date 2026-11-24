package board

import (
	"fmt"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
	"github.com/git-bug/git-bug/util/timestamp"
)

// Reference designates an entity by its namespace and its id: a bug by the
// namespace of the bugs and its id. The entity may be absent from the repository,
// or of a type this client doesn't know: the reference is kept as it is, and
// resolved by whoever displays it.
type Reference struct {
	Namespace string
	Id        entity.Id
}

// IsZero tells whether the reference designates nothing.
func (ref Reference) IsZero() bool {
	return ref == Reference{}
}

// Validate checks that the reference is well-formed, not that the entity exists.
func (ref Reference) Validate() error {
	if text.Empty(ref.Namespace) {
		return fmt.Errorf("namespace is empty")
	}
	if !text.SafeOneLine(ref.Namespace) {
		return fmt.Errorf("namespace has unsafe characters")
	}
	if err := ref.Id.Validate(); err != nil {
		return fmt.Errorf("entity: %w", err)
	}
	return nil
}

// Item is one entry of a board: a draft or a reference to an entity, in a column
// or in none, at a place in the order of the board.
type Item struct {
	// Id is the id of the operation that added the item. A draft converted into an
	// entity keeps it. Operations designate a draft by it, and an item referring to
	// an entity by the id of that entity.
	Id entity.Id
	// CombinedId identifies the item on its own, outside of its board: Id combined
	// with the id of the board.
	CombinedId entity.CombinedId

	// Author is who added the item.
	Author identity.Interface
	// UnixTime is when the item was added.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime timestamp.Timestamp

	// Title is the title of the draft that the item holds. It is empty for an item
	// referring to an entity.
	Title string
	// Entity is the entity that the item refers to. It is zero for a draft.
	Entity Reference

	// Column is the id of the column of the item, empty if it has none: the item is
	// then on the board without being placed in it.
	Column entity.Id
	// Position places the item in the order of the board, the same whatever the
	// column, see util/fracindex.
	Position string
}

// IsDraft tells whether the item holds a draft, instead of referring to an entity.
func (item Item) IsDraft() bool {
	return item.Entity.IsZero()
}
