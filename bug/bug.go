package bug

import (
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util"
	"github.com/kevinburke/go.uuid"
)

const BugsRefPattern = "refs/bugs/"

// Bug hold the data of a bug thread, organized in a way close to
// how it will be persisted inside Git. This is the datastructure
// used for merge of two different version.
type Bug struct {
	// Id used as unique identifier
	id uuid.UUID

	lastCommit util.Hash

	// TODO: need a way to order bugs, probably a Lamport clock

	packs []OperationPack

	staging OperationPack
}

// Create a new Bug
func NewBug() (*Bug, error) {
	// Creating UUID Version 4
	id, err := uuid.ID4()

	if err != nil {
		return nil, err
	}

	return &Bug{
		id:         id,
		lastCommit: "",
	}, nil
}

// IsValid check if the Bug data is valid
func (bug *Bug) IsValid() bool {
	// non-empty
	if len(bug.packs) == 0 && bug.staging.IsEmpty() {
		return false
	}

	// check if each pack is valid
	for _, pack := range bug.packs {
		if !pack.IsValid() {
			return false
		}
	}

	// check if staging is valid if needed
	if !bug.staging.IsEmpty() {
		if !bug.staging.IsValid() {
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
	bug.staging.Append(op)
}

// Write the staging area in Git move the operations to the packs
func (bug *Bug) Commit(repo repository.Repo) error {
	if bug.staging.IsEmpty() {
		return nil
	}

	// Write the Ops as a Git blob containing the serialized array
	hash, err := bug.staging.Write(repo)
	if err != nil {
		return err
	}

	// Write a Git tree referencing this blob
	hash, err = repo.StoreTree(map[string]util.Hash{
		"ops": hash,
	})
	if err != nil {
		return err
	}

	// Write a Git commit referencing the tree, with the previous commit as parent
	if bug.lastCommit != "" {
		hash, err = repo.StoreCommitWithParent(hash, bug.lastCommit)
	} else {
		hash, err = repo.StoreCommit(hash)
	}

	if err != nil {
		return err
	}

	// Create or update the Git reference for this bug
	ref := fmt.Sprintf("%s%s", BugsRefPattern, bug.id.String())
	err = repo.UpdateRef(ref, hash)

	if err != nil {
		return err
	}

	bug.packs = append(bug.packs, bug.staging)
	bug.staging = OperationPack{}

	return nil
}

func (bug *Bug) HumanId() string {
	return bug.id.String()
}

func (bug *Bug) firstOp() Operation {
	for _, pack := range bug.packs {
		for _, op := range pack.Operations {
			return op
		}
	}

	if !bug.staging.IsEmpty() {
		return bug.staging.Operations[0]
	}

	return nil
}
