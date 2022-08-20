package bug

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

type Interface interface {
	// Id returns the Bug identifier
	Id() entity.Id

	// Validate checks if the Bug data is valid
	Validate() error

	// Append an operation into the staging area, to be committed later
	Append(op Operation)

	// Operations returns the ordered operations
	Operations() []Operation

	// NeedCommit indicates that the in-memory state changed and need to be commit in the repository
	NeedCommit() bool

	// Commit writes the staging area in Git and move the operations to the packs
	Commit(repo repository.ClockedRepo) error

	// FirstOp lookup for the very first operation of the bug.
	// For a valid Bug, this operation should be a CreateOp
	FirstOp() Operation

	// LastOp lookup for the very last operation of the bug.
	// For a valid Bug, should never be nil
	LastOp() Operation

	// Compile a bug in an easily usable snapshot
	Compile() *Snapshot

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}
