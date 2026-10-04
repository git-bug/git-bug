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

// Tracked is an Entity seen as its committed state: what is in the repository,
// written to and reloaded from there without going through a staging area. It is
// what can be shared between several holders staging operations of their own on
// top of it, see Mutable.
type Tracked[SnapT Snapshot, OpT Operation] interface {
	entity.Interface

	// Compile an Entity in an easily usable snapshot
	Compile() SnapT

	// CommitOperations writes the given operations in Git, on top of the committed
	// state, leaving the staging area untouched
	CommitOperations(repo repository.ClockedRepo, ops []OpT) error

	// Repair reloads the committed state from the repository, typically after a
	// commit failed with repository.ErrRefChanged, leaving the staging area untouched
	Repair(repo repository.ClockedRepo, resolvers entity.Resolvers) error

	// LastCommit returns the hash of the commit holding the last committed operations,
	// that is what the Entity's reference points to as far as the Entity knows.
	// It is empty if the Entity has never been committed.
	LastCommit() repository.Hash

	// CreateLamportTime return the Lamport time of creation
	CreateLamportTime() lamport.Time

	// EditLamportTime return the Lamport time of the last edit
	EditLamportTime() lamport.Time
}
