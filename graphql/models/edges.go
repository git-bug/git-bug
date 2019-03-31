package models

// GetCursor return the cursor entry of an edge
func (e OperationEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e BugEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e CommentEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e TimelineItemEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e IdentityEdge) GetCursor() string {
	return e.Cursor
}
