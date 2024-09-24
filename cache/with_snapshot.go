package cache

import (
	"sync"

	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

var _ dag.ReadWrite[dag.Snapshot, dag.OperationWithApply[dag.Snapshot]] = &withSnapshot[dag.Snapshot, dag.OperationWithApply[dag.Snapshot]]{}

// withSnapshot encapsulate an entity and maintain a snapshot efficiently.
type withSnapshot[SnapT dag.Snapshot, OpT dag.OperationWithApply[SnapT]] struct {
	dag.ReadWrite[SnapT, OpT]
	mu   sync.Mutex
	snap *SnapT
}

func newWithSnapshot[SnapT dag.Snapshot, OpT dag.OperationWithApply[SnapT]](readWrite dag.ReadWrite[SnapT, OpT]) *withSnapshot[SnapT, OpT] {
	return &withSnapshot[SnapT, OpT]{ReadWrite: readWrite}
}

func (ws *withSnapshot[SnapT, OpT]) Snapshot() SnapT {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.snap == nil {
		snap := ws.ReadWrite.Snapshot()
		ws.snap = &snap
	}
	return *ws.snap
}

// Append intercept Bug.Append() to update the snapshot efficiently
func (ws *withSnapshot[SnapT, OpT]) Append(op OpT) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.ReadWrite.Append(op)

	if ws.snap == nil {
		return
	}

	op.Apply(*ws.snap)
	(*ws.snap).AppendOperation(op)
}

// Commit intercept Bug.Commit() to update the snapshot efficiently
func (ws *withSnapshot[SnapT, OpT]) Commit(repo repository.ClockedRepo) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	err := ws.ReadWrite.Commit(repo)
	if err != nil {
		ws.snap = nil
		return err
	}

	return nil
}
