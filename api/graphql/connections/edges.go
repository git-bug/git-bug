package connections

import "github.com/git-bug/git-bug/entity"

// CursorEdge is a minimal edge carrying only a cursor. Use it with
// connections.Connection when the edge type needs no additional fields.
type CursorEdge struct {
	Cursor string
}

func (e CursorEdge) GetCursor() string { return e.Cursor }

// LazyBugEdge is a special relay edge used to implement a lazy loading connection
type LazyBugEdge struct {
	Id     entity.Id
	Cursor string
}

// GetCursor return the cursor of a LazyBugEdge
func (lbe LazyBugEdge) GetCursor() string {
	return lbe.Cursor
}

// LazyIdentityEdge is a special relay edge used to implement a lazy loading connection
type LazyIdentityEdge struct {
	Id     entity.Id
	Cursor string
}

// GetCursor return the cursor of a LazyIdentityEdge
func (lbe LazyIdentityEdge) GetCursor() string {
	return lbe.Cursor
}
