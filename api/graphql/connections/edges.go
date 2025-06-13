package connections

import "github.com/git-bug/git-bug/entity"

// LazyBoardEdge is a special relay edge used to implement a lazy loading connection
type LazyBoardEdge struct {
	Id     entity.Id
	Cursor string
}

// GetCursor return the cursor of a LazyBoardEdge
func (lbe LazyBoardEdge) GetCursor() string {
	return lbe.Cursor
}

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
