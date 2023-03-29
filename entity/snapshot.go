package entity

// Snapshot is the minimal interface that a snapshot need to implement
type Snapshot interface {
	// AllOperations returns all the operations that have been applied to that snapshot, in order
	AllOperations() []Operation
	// AppendOperation add an operation in the list
	AppendOperation(op Operation)
}

type CompileToSnapshot[SnapT Snapshot] interface {
	// Compile an Entity in an easily usable snapshot
	Compile() SnapT
}
