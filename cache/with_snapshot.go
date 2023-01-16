package cache

import (
	"sync"

	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var _ dag.Interface[dag.Snapshot, dag.OperationWithApply[dag.Snapshot]] = &withSnapshot[dag.Snapshot, dag.OperationWithApply[dag.Snapshot]]{}

// withSnapshot encapsulate an entity and maintain a snapshot efficiently.
type withSnapshot[SnapT dag.Snapshot, OpT dag.OperationWithApply[SnapT]] struct {
	dag.Interface[SnapT, OpT]
	mu   sync.Mutex
	snap *SnapT
}

func (ws *withSnapshot[SnapT, OpT]) Compile() SnapT {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.snap == nil {
		snap := ws.Interface.Compile()
		ws.snap = &snap
	}
	return *ws.snap
}

// Append intercept Bug.Append() to update the snapshot efficiently
func (ws *withSnapshot[SnapT, OpT]) Append(op OpT) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.Interface.Append(op)

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

	err := ws.Interface.Commit(repo)
	if err != nil {
		ws.snap = nil
		return err
	}

	return nil
}
