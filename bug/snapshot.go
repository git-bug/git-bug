package bug

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	Title    string
	Comments []Comment
	Labels   []Label
}
