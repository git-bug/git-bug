package connections

// LazyIdentityEdge is a special relay edge used to implement a lazy loading connection
type LazyIdentityEdge struct {
	Id     string
	Cursor string
}

// GetCursor return the cursor of a LazyIdentityEdge
func (lbe LazyIdentityEdge) GetCursor() string {
	return lbe.Cursor
}
