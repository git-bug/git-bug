package models

// GetCursor return the cursor entry of an edge
func (e OperationEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e BoardColumnEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e BoardItemEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e BugEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e BugCommentEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e BugTimelineItemEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e IdentityEdge) GetCursor() string {
	return e.Cursor
}

// GetCursor return the cursor entry of an edge
func (e LabelEdge) GetCursor() string {
	return e.Cursor
}
