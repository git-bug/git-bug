package bug

// OperationPack represent an ordered set of operation to apply
// to a Bug. These operations are stored in a single Git commit.
//
// These commits will be linked together in a linear chain of commits
// inside Git to form the complete ordered chain of operation to
// apply to get the final state of the Bug
type OperationPack struct {
	Operations []Operation
}

// Append a new operation to the pack
func (opp *OperationPack) Append(op Operation) {
	opp.Operations = append(opp.Operations, op)
}

func (opp *OperationPack) IsEmpty() bool {
	return len(opp.Operations) == 0
}

func (opp *OperationPack) IsValid() bool {
	return !opp.IsEmpty()
}
