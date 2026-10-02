package dag

import (
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// Mutable is what creating an operation needs from an Entity: the state to
// create the operation against, and where to stage it. Besides an Entity,
// anything holding staged operations of its own on top of one can implement it.
type Mutable[SnapT Snapshot, OpT Operation] interface {
	// Id return the Entity identifier
	Id() entity.Id

	// Append an operation into the staging area, to be committed later
	Append(op OpT)

	// Compile an Entity in an easily usable snapshot, staged operations included
	Compile() SnapT
}

// TODO: rename?
// Interface define the extended interface of a dag.Entity
type Interface[SnapT Snapshot, OpT Operation] interface {
	entity.Interface
	Mutable[SnapT, OpT]

	// NeedCommit indicates that the in-memory state changed and need to be committed in the repository
	NeedCommit() bool

	// Commit writes the staging area in Git and move the operations to the packs
	Commit(repo repository.ClockedRepo) error

	// CommitOperations writes the given operations in Git, on top of the committed
	// state, leaving the staging area untouched
	CommitOperations(repo repository.ClockedRepo, ops []OpT) error

	// LastCommit returns the hash of the commit holding the last committed operations,
	// that is what the Entity's reference points to as far as the Entity knows.
	// It is empty if the Entity has never been committed.
	LastCommit() repository.Hash

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}
