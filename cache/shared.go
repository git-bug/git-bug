package cache

import (
	"sync"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// sharedEntity is the single copy of an entity that the cache keeps in memory: its
// committed state, as found in the repository, with its snapshot memoized. It holds
// no pending operations. Those belong to the views handed out to callers, see
// CachedEntityBase, which write through it, and reload it when the reference moved.
type sharedEntity[SnapT dag.Snapshot, OpT dag.Operation] struct {
	dag.Tracked[SnapT, OpT]

	// mu serializes the changes to the committed state with the snapshot memo, so
	// that the memo, LastCommit and the clocks always describe the same state.
	mu   sync.RWMutex
	snap *SnapT
}

// Compile returns the snapshot of the committed state. It is shared with every
// caller, and must not be modified.
func (s *sharedEntity[SnapT, OpT]) Compile() SnapT {
	snap, _ := s.state()
	return snap
}

// state returns the snapshot of the committed state, along with the commit it was
// built from.
func (s *sharedEntity[SnapT, OpT]) state() (SnapT, repository.Hash) {
	s.mu.RLock()
	if s.snap != nil {
		defer s.mu.RUnlock()
		return *s.snap, s.Tracked.LastCommit()
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snap == nil {
		snap := s.Tracked.Compile()
		s.snap = &snap
	}
	return *s.snap, s.Tracked.LastCommit()
}

// CommitOperations writes the given operations on top of the committed state, see
// dag.Tracked. The memoized snapshot is rebuilt on the next read.
func (s *sharedEntity[SnapT, OpT]) CommitOperations(repo repository.ClockedRepo, ops []OpT) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.Tracked.CommitOperations(repo, ops)
	if err == nil {
		s.snap = nil
	}
	return err
}

// Repair reloads the committed state from the repository, see dag.Tracked. The
// memoized snapshot is rebuilt on the next read.
func (s *sharedEntity[SnapT, OpT]) Repair(repo repository.ClockedRepo, resolvers entity.Resolvers) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap = nil
	return s.Tracked.Repair(repo, resolvers)
}
