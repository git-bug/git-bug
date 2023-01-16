package dag

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// Interface define the extended interface of a dag.Entity
type Interface[SnapT Snapshot, OpT Operation] interface {
	entity.Interface

	// Validate checks if the Entity data is valid
	Validate() error

	// Append an operation into the staging area, to be committed later
	Append(op OpT)

	// Operations returns the ordered operations
	Operations() []OpT

	// NeedCommit indicates that the in-memory state changed and need to be committed in the repository
	NeedCommit() bool

	// Commit writes the staging area in Git and move the operations to the packs
	Commit(repo repository.ClockedRepo) error

	// CommitAsNeeded execute a Commit only if necessary. This function is useful to avoid getting an error if the Entity
	// is already in sync with the repository.
	CommitAsNeeded(repo repository.ClockedRepo) error

	// FirstOp lookup for the very first operation of the Entity.
	FirstOp() OpT

	// LastOp lookup for the very last operation of the Entity.
	// For a valid Entity, should never be nil
	LastOp() OpT

	// Compile an Entity in an easily usable snapshot
	Compile() SnapT

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}
