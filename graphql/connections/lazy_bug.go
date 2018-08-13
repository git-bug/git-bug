package connections

// LazyBugEdge is a special relay edge used to implement a lazy loading connection
type LazyBugEdge struct {
	Id     string
	Cursor string
}

// GetCursor return the cursor of a LazyBugEdge
func (lbe LazyBugEdge) GetCursor() string {
	return lbe.Cursor
}
