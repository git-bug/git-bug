package bug

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

type Interface interface {
	// Id return the Bug identifier
	Id() entity.Id

	// Validate check if the Bug data is valid
	Validate() error

	// Append an operation into the staging area, to be committed later
	Append(op Operation)

	// Operations return the ordered operations
	Operations() []Operation

	// Indicate that the in-memory state changed and need to be commit in the repository
	NeedCommit() bool

	// Commit write the staging area in Git and move the operations to the packs
	Commit(repo repository.ClockedRepo) error

	// Lookup for the very first operation of the bug.
	// For a valid Bug, this operation should be a CreateOp
	FirstOp() Operation

	// Lookup for the very last operation of the bug.
	// For a valid Bug, should never be nil
	LastOp() Operation

	// Compile a bug in a easily usable snapshot
	Compile() Snapshot

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}

func bugFromInterface(bug Interface) *Bug {
	switch bug := bug.(type) {
	case *Bug:
		return bug
	case *WithSnapshot:
		return bug.Bug
	default:
		panic("missing type case")
	}
}
