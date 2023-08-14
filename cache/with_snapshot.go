package cache

import (
	"sync"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

var _ entity.Interface[entity.Snapshot, dag.OperationWithApply[entity.Snapshot]] = &withSnapshot[entity.Snapshot, dag.OperationWithApply[entity.Snapshot]]{}

// withSnapshot encapsulate an entity and maintain a snapshot efficiently.
type withSnapshot[SnapT entity.Snapshot, OpT dag.OperationWithApply[SnapT]] struct {
	entity.WithCommit[SnapT, OpT]
	mu   sync.Mutex
	snap *SnapT
}

func (ws *withSnapshot[SnapT, OpT]) Compile() SnapT {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.snap == nil {
		snap := ws.WithCommit.Compile()
		ws.snap = &snap
	}
	return *ws.snap
}

// Append intercept Bug.Append() to update the snapshot efficiently
func (ws *withSnapshot[SnapT, OpT]) Append(op OpT) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.WithCommit.Append(op)

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

	err := ws.WithCommit.Commit(repo)
	if err != nil {
		ws.snap = nil
		return err
	}

	return nil
}
