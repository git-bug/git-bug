package cache

import (
	"sync"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// type withSnapshot[SnapT dag.Snapshot, OpT dag.OperationWithApply[SnapT]] struct {
// 	dag.Interface[SnapT, OpT]
// 	snap dag.Snapshot
// }
//
//
// func (ws *withSnapshot[SnapT, OpT]) Compile() dag.Snapshot {
// 	if ws.snap == nil {
// 		snap := ws.Interface.Compile()
// 		ws.snap = snap
// 	}
// 	return ws.snap
// }
//
// // Append intercept Bug.Append() to update the snapshot efficiently
// func (ws *withSnapshot[SnapT, OpT]) Append(op OpT) {
// 	ws.Interface.Append(op)
//
// 	if ws.snap == nil {
// 		return
// 	}
//
// 	op.Apply(ws.snap)
// 	ws.snap. = append(ws.snap.Operations, op)
// }
//
// // Commit intercept Bug.Commit() to update the snapshot efficiently
// func (ws *withSnapshot[SnapT, OpT]) Commit(repo repository.ClockedRepo) error {
// 	err := ws.Interface.Commit(repo)
//
// 	if err != nil {
// 		ws.snap = nil
// 		return err
// 	}
//
// 	// Commit() shouldn't change anything of the bug state apart from the
// 	// initial ID set
//
// 	if ws.snap == nil {
// 		return nil
// 	}
//
// 	ws.snap.id = ws.Interface.Id()
// 	return nil
// }

type CachedEntityBase[SnapT dag.Snapshot, OpT dag.Operation] struct {
	repo            repository.ClockedRepo
	entityUpdated   func(id entity.Id) error
	getUserIdentity getUserIdentityFunc

	mu     sync.RWMutex
	entity dag.Interface[SnapT, OpT]
}

func (e *CachedEntityBase[SnapT, OpT]) Id() entity.Id {
	return e.entity.Id()
}

func (e *CachedEntityBase[SnapT, OpT]) Snapshot() SnapT {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.Compile()
}

func (e *CachedEntityBase[SnapT, OpT]) notifyUpdated() error {
	return e.entityUpdated(e.entity.Id())
}

// ResolveOperationWithMetadata will find an operation that has the matching metadata
func (e *CachedEntityBase[SnapT, OpT]) ResolveOperationWithMetadata(key string, value string) (entity.Id, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	// preallocate but empty
	matching := make([]entity.Id, 0, 5)

	for _, op := range e.entity.Operations() {
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

func (e *CachedEntityBase[SnapT, OpT]) Validate() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.Validate()
}

func (e *CachedEntityBase[SnapT, OpT]) Commit() error {
	e.mu.Lock()
	err := e.entity.Commit(e.repo)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	return e.notifyUpdated()
}

func (e *CachedEntityBase[SnapT, OpT]) CommitAsNeeded() error {
	e.mu.Lock()
	err := e.entity.CommitAsNeeded(e.repo)
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()
	return e.notifyUpdated()
}

func (e *CachedEntityBase[SnapT, OpT]) NeedCommit() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.entity.NeedCommit()
}

func (e *CachedEntityBase[SnapT, OpT]) CreateLamportTime() lamport.Time {
	return e.entity.CreateLamportTime()
}

func (e *CachedEntityBase[SnapT, OpT]) EditLamportTime() lamport.Time {
	return e.entity.EditLamportTime()
}

func (e *CachedEntityBase[SnapT, OpT]) FirstOp() OpT {
	return e.entity.FirstOp()
}
