package cache

import (
	"context"
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

	// depends, if not nil, returns what the snapshot depends on besides the
	// committed state, such as the config judging the operations of a bug, as a
	// comparable value: the memoized snapshot is rebuilt when it changes.
	depends func() any

	// mu serializes the changes to the committed state with the snapshot memo, so
	// that the memo, LastCommit and the clocks always describe the same state.
	mu   sync.RWMutex
	snap *SnapT
	// dependency is the value of depends when the memo was built
	dependency any
	// generation counts the memos built, see state
	generation uint64
}

// Compile returns the snapshot of the committed state. It is shared with every
// caller, and must not be modified.
func (s *sharedEntity[SnapT, OpT]) Compile() SnapT {
	snap, _ := s.state()
	return snap
}

// state returns the snapshot of the committed state, along with the generation of
// the memo, which changes whenever the snapshot is rebuilt: because the committed
// state moved, or what it depends on changed, see depends.
func (s *sharedEntity[SnapT, OpT]) state() (SnapT, uint64) {
	// read before compiling: should it change meanwhile, the next call compiles
	// again, the memo being recorded with the earlier value
	var dependency any
	if s.depends != nil {
		dependency = s.depends()
	}

	s.mu.RLock()
	if s.snap != nil && s.dependency == dependency {
		defer s.mu.RUnlock()
		return *s.snap, s.generation
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snap == nil || s.dependency != dependency {
		snap := s.Tracked.Compile()
		s.snap, s.dependency = &snap, dependency
		s.generation++
	}
	return *s.snap, s.generation
}

// CommitOperations writes the given operations on top of the committed state, see
// dag.Tracked. The memoized snapshot is rebuilt on the next read.
//
// TODO: the rebuild replays the whole history. It could be avoided: when the commit
// landed on the base a view compiled against, that view's snapshot is the new
// committed state and could be taken as is; otherwise, the previous snapshot could
// be cloned and the new operations applied to it. Same for Repair's incremental path.
func (s *sharedEntity[SnapT, OpT]) CommitOperations(ctx context.Context, repo repository.Repo, ops []OpT) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.Tracked.CommitOperations(ctx, repo, ops)
	if err == nil {
		s.snap = nil
	}
	return err
}

// Repair reloads the committed state from the repository, see dag.Tracked. The
// memoized snapshot is rebuilt on the next read if the committed state moved: a
// failed Repair leaves it as it was, as does one finding nothing new, so that the
// snapshot handed out stays the same, for what depends on it, see depends.
func (s *sharedEntity[SnapT, OpT]) Repair(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.Tracked.LastCommit()
	err := s.Tracked.Repair(ctx, repo, resolvers)
	if s.Tracked.LastCommit() != before {
		s.snap = nil
	}
	return err
}
