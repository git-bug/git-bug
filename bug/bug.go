package bug

import (
	"github.com/kevinburke/go.uuid"
)

// Bug hold the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the datastructure
// used for merge of two different version.
type Bug struct {
	// Id used as unique identifier
	Id uuid.UUID

	// TODO: need a way to order bugs
	// Probably a Lamport clock

	Packs []OperationPack

	Staging OperationPack
}

// Create a new Bug
func NewBug() (*Bug, error) {

	// Creating UUID Version 4
	id, err := uuid.ID4()

	if err != nil {
		return nil, err
	}

	return &Bug{
		Id: id,
	}, nil
}

// IsValid check if the Bug data is valid
func (bug *Bug) IsValid() bool {
	// non-empty
	if len(bug.Packs) == 0 && bug.Staging.IsEmpty() {
		return false
	}

	// check if each pack is valid
	for _, pack := range bug.Packs {
		if !pack.IsValid() {
			return false
		}
	}

	// check if Staging is valid if needed
	if !bug.Staging.IsEmpty() {
		if !bug.Staging.IsValid() {
			return false
		}
	}

	// The very first Op should be a CREATE
	firstOp := bug.firstOp()
	if firstOp == nil || firstOp.OpType() != CREATE {
		return false
	}

	// Check that there is no more CREATE op
	it := NewOperationIterator(bug)
	createCount := 0
	for it.Next() {
		if it.Value().OpType() == CREATE {
			createCount++
		}
	}

	if createCount != 1 {
		return false
	}

	return true
}

func (bug *Bug) Append(op Operation) {
	bug.Staging.Append(op)
}

func (bug *Bug) Commit() {
	bug.Packs = append(bug.Packs, bug.Staging)
	bug.Staging = OperationPack{}
}

func (bug *Bug) HumanId() string {
	return bug.Id.String()
}

func (bug *Bug) firstOp() Operation {
	for _, pack := range bug.Packs {
		for _, op := range pack.Operations {
			return op
		}
	}

	if !bug.Staging.IsEmpty() {
		return bug.Staging.Operations[0]
	}

	return nil
}
