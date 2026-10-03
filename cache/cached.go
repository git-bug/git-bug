package cache

import (
	"errors"
	"fmt"
	"sync"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// maxCommitRetries bounds how many times a commit rejected because the reference
// moved is written again on the reloaded state. Every retry means another writer
// made progress, so the bound only guards against a misbehaving one.
const maxCommitRetries = 5

// cloneable is a snapshot that can be copied, for operations to be applied to the
// copy without the original noticing.
type cloneable[SnapT any] interface {
	dag.Snapshot
	Clone() SnapT
}

// CachedEntityBase provide the base function of an entity managed by the cache: a
// view of the single loaded copy of the entity, see sharedEntity, with operations
// of its own staged on top of it. It is what creating an operation works on, see
// dag.Mutable.
//
// Each caller gets its own view, see SubCache.Resolve. What a view stages is
// invisible to the others until Commit writes it through the shared copy, in a
// single pass. Once Commit returned, successfully or not, the staged operations
// are gone: a failure can't be saved by a later commit.
//
// A view is safe for concurrent use: its methods can be called from any
// goroutine, and a snapshot it handed out is never modified afterwards. It is
// not a transaction: two operations created concurrently on the same view each
// see the state before the other, as they would on two views.
type CachedEntityBase[SnapT cloneable[SnapT], OpT dag.OperationWithApply[SnapT]] struct {
	repo            repository.ClockedRepo
	resolvers       func() entity.Resolvers
	onCommit        func() error // called after each commit
	getUserIdentity getUserIdentityFunc

	shared *sharedEntity[SnapT, OpT]

	mu     sync.Mutex
	staged []OpT
	// snap is the shared snapshot as of the commit base, with the staged
	// operations applied. It is nil when nothing is staged, or when it has to be
	// rebuilt because an operation was staged or the shared copy moved.
	snap *SnapT
	base repository.Hash
}

func (e *CachedEntityBase[SnapT, OpT]) Id() entity.Id {
	return e.shared.Id()
}

// Snapshot returns the state of the entity, staged operations included, see Compile.
func (e *CachedEntityBase[SnapT, OpT]) Snapshot() SnapT {
	return e.Compile()
}

// Compile returns the state of the entity, staged operations included. With none,
// it is the shared snapshot of the committed state, which must not be modified.
// Otherwise, it is a copy of it with the staged operations applied, rebuilt when
// the committed state moved.
func (e *CachedEntityBase[SnapT, OpT]) Compile() SnapT {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.staged) == 0 {
		return e.shared.Compile()
	}

	committed, base := e.shared.state()
	if e.snap != nil && e.base == base {
		return *e.snap
	}

	snap := committed.Clone()
	for _, op := range e.staged {
		op.Apply(snap)
		snap.AppendOperation(op)
	}
	e.snap, e.base = &snap, base
	return snap
}

// Append stages an operation, to be committed later.
func (e *CachedEntityBase[SnapT, OpT]) Append(op OpT) {
	// an operation's id is computed on first use: fix it while the operation is
	// still the caller's alone
	_ = op.Id()

	e.mu.Lock()
	defer e.mu.Unlock()
	e.staged = append(e.staged, op)
	// snapshots handed out are never modified: the next Compile builds a new one
	e.snap = nil
}

// ResolveOperationWithMetadata will find an operation that has the matching metadata
func (e *CachedEntityBase[SnapT, OpT]) ResolveOperationWithMetadata(key string, value string) (entity.Id, error) {
	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	// Metadata can be added to an operation by a SetMetadataOperation, which only
	// takes effect when applied. Go through the snapshot to make sure it has been.
	for _, op := range e.Snapshot().AllOperations() {
		opValue, ok := op.GetMetadata(key)
		if ok && value == opValue {
			matching = append(matching, op.Id())
		}
	}

	if len(matching) == 0 {
		return "", ErrNoMatchingOp
	}

	if len(matching) > 1 {
		return "", entity.NewErrMultipleMatch("operation", matching)
	}

	return matching[0], nil
}

// Commit writes the staged operations in the repository, on top of the committed
// state. If the reference moved in the meantime, the shared copy is reloaded and
// the operations written again on top of what moved it: operations commute, so a
// write that raced with another writer only has to be redone, never reconciled.
// Whether it succeeded or not, the staged operations are dropped.
func (e *CachedEntityBase[SnapT, OpT]) Commit() error {
	return e.commit(true)
}

// CommitAsNeeded execute a Commit only if necessary. This function is useful to
// avoid getting an error if the view has nothing staged.
func (e *CachedEntityBase[SnapT, OpT]) CommitAsNeeded() error {
	return e.commit(false)
}

// commit is Commit, with nothing staged being an error only if required.
func (e *CachedEntityBase[SnapT, OpT]) commit(required bool) error {
	e.mu.Lock()
	if len(e.staged) == 0 {
		e.mu.Unlock()
		if required {
			return fmt.Errorf("can't commit an entity with no pending operation")
		}
		return nil
	}
	err := e.commitLocked()
	e.mu.Unlock()
	if err != nil {
		return err
	}
	return e.onCommit()
}

func (e *CachedEntityBase[SnapT, OpT]) commitLocked() error {
	staged := e.staged
	e.staged, e.snap = nil, nil

	for attempt := 0; ; attempt++ {
		err := e.shared.CommitOperations(e.repo, staged)
		if !errors.Is(err, repository.ErrRefChanged) || attempt == maxCommitRetries {
			return err
		}
		err = e.shared.Repair(e.repo, e.resolvers())
		if err != nil {
			return err
		}
	}
}

// NeedCommit indicate if the view holds staged operations, that need to be committed
// in the repository.
func (e *CachedEntityBase[SnapT, OpT]) NeedCommit() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.staged) > 0
}

// LastCommit returns the hash of the commit holding the committed state of the entity.
func (e *CachedEntityBase[SnapT, OpT]) LastCommit() repository.Hash {
	return e.shared.LastCommit()
}

func (e *CachedEntityBase[SnapT, OpT]) CreateLamportTime() lamport.Time {
	return e.shared.CreateLamportTime()
}

func (e *CachedEntityBase[SnapT, OpT]) EditLamportTime() lamport.Time {
	return e.shared.EditLamportTime()
}
