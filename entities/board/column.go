package board

import (
	"github.com/git-bug/git-bug/entity"
)

// Column is a named place of a board, in which items sit. It belongs to its
// board: the columns of two boards are unrelated, even under the same name.
type Column struct {
	// Id is the id of the operation that added the column. Items and operations
	// refer to the column by it, so that renaming a column changes no item.
	Id entity.Id
	// CombinedId identifies the column on its own, outside of its board: Id
	// combined with the id of the board.
	CombinedId entity.CombinedId

	// Name is not unique: two columns of a board may have the same.
	Name string
	// Position places the column among the others, see util/fracindex.
	Position string
	// PriorityOrdered tells whether the order of the items of the column is their
	// priority, the first one being the one to take first: typically a backlog.
	// In the other columns the order says nothing.
	PriorityOrdered bool
}
