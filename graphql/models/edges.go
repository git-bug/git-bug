package models

func (e OperationEdge) GetCursor() string {
	return e.Cursor
}

func (e BugEdge) GetCursor() string {
	return e.Cursor
}

func (e CommentEdge) GetCursor() string {
	return e.Cursor
}
