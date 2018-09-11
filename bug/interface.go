package bug

import (
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

type Interface interface {
	// Id return the Bug identifier
	Id() string

	// HumanId return the Bug identifier truncated for human consumption
	HumanId() string

	// IsValid check if the Bug data is valid
	IsValid() bool

	// Append an operation into the staging area, to be committed later
	Append(op Operation)

	// Append an operation into the staging area, to be committed later
	HasPendingOp() bool

	// Commit write the staging area in Git and move the operations to the packs
	Commit(repo repository.Repo) error

	// Merge a different version of the same bug by rebasing operations of this bug
	// that are not present in the other on top of the chain of operations of the
	// other version.
	Merge(repo repository.Repo, other Interface) (bool, error)

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
	switch bug.(type) {
	case *Bug:
		return bug.(*Bug)
	case *WithSnapshot:
		return bug.(*WithSnapshot).Bug
	default:
		panic("missing type case")
	}
}
