package bug

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	Title    string
	Comments []Comment
	Labels   []Label
}

//func (bug Bug) Check() error {
//	if bug.Operations.Len() == 0 {
//		return "Empty operation log"
//	}
//
//	if bug.Operations.Elems()
//
//	return true
//}
