package connections

type LazyBugEdge struct {
	Id     string
	Cursor string
}

func (lbe LazyBugEdge) GetCursor() string {
	return lbe.Cursor
}
