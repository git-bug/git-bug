package dag

import (
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

type CompileTo[SnapT Snapshot] interface {
	// Snapshot compiles an Entity in an easily usable snapshot
	Snapshot() SnapT
}

// ReadOnly defines the extended read-only interface of a dag.Entity
type ReadOnly[SnapT Snapshot, OpT Operation] interface {
	entity.Interface

	CompileTo[SnapT]

	// NeedCommit indicates that the in-memory state changed and need to be committed in the repository
	NeedCommit() bool

	// FirstOp lookup for the very first operation of the Entity.
	FirstOp() OpT

	// LastOp lookup for the very last operation of the Entity.
	// For a valid Entity, it should never be nil.
	LastOp() OpT

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}

// ReadWrite is an entity interface that includes the direct manipulation of operations.
type ReadWrite[SnapT Snapshot, OpT Operation] interface {
	ReadOnly[SnapT, OpT]

	// Commit writes the staging area in Git and move the operations to the packs
	Commit(repo repository.ClockedRepo) error

	// CommitAsNeeded execute a Commit only if necessary. This function is useful to avoid getting an error if the Entity
	// is already in sync with the repository.
	CommitAsNeeded(repo repository.ClockedRepo) error

	// Append an operation into the staging area, to be committed later
	Append(op OpT)

	// Operations return the ordered operations
	Operations() []OpT
}
