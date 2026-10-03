package dag

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

func TestWriteRead(t *testing.T) {
	repo, id1, id2, resolver, def := makeTestContext()

	entity := wrapper(New(def))
	require.False(t, entity.NeedCommit())

	entity.Append(newOp1(id1, "foo"))
	entity.Append(newOp2(id1, "bar"))

	require.True(t, entity.NeedCommit())
	require.NoError(t, entity.CommitAsNeeded(repo))
	require.False(t, entity.NeedCommit())

	entity.Append(newOp2(id2, "foobar"))
	require.True(t, entity.NeedCommit())
	require.NoError(t, entity.CommitAsNeeded(repo))
	require.False(t, entity.NeedCommit())

	read, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)

	assertEqualEntities(t, entity.Entity, read.Entity)
}

func TestWriteReadMultipleAuthor(t *testing.T) {
	repo, id1, id2, resolver, def := makeTestContext()

	entity := wrapper(New(def))

	entity.Append(newOp1(id1, "foo"))
	entity.Append(newOp2(id2, "bar"))

	require.NoError(t, entity.CommitAsNeeded(repo))

	entity.Append(newOp2(id1, "foobar"))
	require.NoError(t, entity.CommitAsNeeded(repo))

	read, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)

	assertEqualEntities(t, entity.Entity, read.Entity)
}

// A ref must point to the Entity it is named after.
func TestReadIdMismatch(t *testing.T) {
	repo, id1, _, resolver, def := makeTestContext()

	e1 := wrapper(New(def))
	e1.Append(newOp1(id1, "foo"))
	require.NoError(t, e1.Commit(repo))

	e2 := wrapper(New(def))
	e2.Append(newOp1(id1, "bar"))
	require.NoError(t, e2.Commit(repo))

	// the ref of e1 now points to the data of e2
	require.NoError(t, repo.UpdateRef(def.Namespace, e1.Id().String(), e1.lastCommit, e2.lastCommit))

	_, err := Read(def, wrapper, repo, resolver, e1.Id())
	require.ErrorContains(t, err, "doesn't match its id")

	var readAllErr error
	for streamed := range ReadAll(def, wrapper, repo, resolver) {
		if streamed.Err != nil {
			readAllErr = streamed.Err
		}
	}
	require.ErrorContains(t, readAllErr, "doesn't match its id")
}

func TestCommitStaleCopy(t *testing.T) {
	repo, id1, _, resolver, def := makeTestContext()

	entity := wrapper(New(def))
	entity.Append(newOp1(id1, "foo"))
	require.NoError(t, entity.Commit(repo))

	copy1, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)
	copy2, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)
	expected, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)

	copy1.Append(newOp2(id1, "first"))
	require.NoError(t, copy1.Commit(repo))

	op := newOp2(id1, "second")
	copy2.Append(op)
	expected.Append(op)
	require.ErrorIs(t, copy2.Commit(repo), repository.ErrRefChanged)

	// the stale copy is left untouched, with its operation still pending
	require.True(t, copy2.NeedCommit())
	assertEqualEntities(t, expected.Entity, copy2.Entity)

	read, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)
	assertEqualEntities(t, copy1.Entity, read.Entity)
}

// LastCommit follows the reference, and only moves with it.
func TestLastCommit(t *testing.T) {
	repo, id1, _, resolver, def := makeTestContext()

	requireRef := func(t *testing.T, e *Foo) {
		t.Helper()
		ref, err := repo.ResolveRef(def.Namespace, e.Id().String())
		require.NoError(t, err)
		require.Equal(t, ref, e.LastCommit())
	}

	entity := wrapper(New(def))
	require.Empty(t, entity.LastCommit())

	entity.Append(newOp1(id1, "foo"))
	require.Empty(t, entity.LastCommit())
	require.NoError(t, entity.Commit(repo))
	requireRef(t, entity)

	read, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)
	requireRef(t, read)

	// pending operations don't move it
	entity.Append(newOp2(id1, "bar"))
	requireRef(t, entity)
	require.NoError(t, entity.Commit(repo))
	requireRef(t, entity)

	// neither does a failed commit
	read.Append(newOp2(id1, "stale"))
	lastCommit := read.LastCommit()
	require.ErrorIs(t, read.Commit(repo), repository.ErrRefChanged)
	require.Equal(t, lastCommit, read.LastCommit())
}

// failingCommitRepo fails the failAt-th call to StoreCommit
type failingCommitRepo struct {
	repository.ClockedRepo
	failAt int
	calls  int
}

func (r *failingCommitRepo) StoreCommit(treeHash repository.Hash, parents ...repository.Hash) (repository.Hash, error) {
	r.calls++
	if r.calls == r.failAt {
		return "", errors.New("store commit failed")
	}
	return r.ClockedRepo.StoreCommit(treeHash, parents...)
}

func TestCommitFailureMidway(t *testing.T) {
	repo, id1, id2, resolver, def := makeTestContext()

	entity := wrapper(New(def))
	entity.Append(newOp1(id1, "foo"))
	require.NoError(t, entity.Commit(repo))

	expected, err := Read(def, wrapper, repo, resolver, entity.Id())
	require.NoError(t, err)

	// two authors means two commits, the second one fails
	op1, op2 := newOp2(id1, "bar"), newOp2(id2, "foobar")
	entity.Append(op1)
	entity.Append(op2)
	expected.Append(op1)
	expected.Append(op2)

	failing := &failingCommitRepo{ClockedRepo: repo, failAt: 2}
	require.Error(t, entity.Commit(failing))
	require.Equal(t, 2, failing.calls)

	require.True(t, entity.NeedCommit())
	assertEqualEntities(t, expected.Entity, entity.Entity)

	require.NoError(t, entity.Commit(repo))
}

// countingRepo counts the calls to ReadCommit
type countingRepo struct {
	repository.ClockedRepo
	reads int
}

func (r *countingRepo) ReadCommit(hash repository.Hash) (repository.Commit, error) {
	r.reads++
	return r.ClockedRepo.ReadCommit(hash)
}

func TestCommitOperations(t *testing.T) {
	t.Run("pending first operation", func(t *testing.T) {
		repo, id1, _, resolver, def := makeTestContext()

		e := wrapper(New(def))
		create := newOp1(id1, "foo")
		e.Append(create)

		require.Error(t, e.CommitOperations(repo, []Operation{newOp2(id1, "bar")}))
		require.Equal(t, []Operation{create}, e.Operations())
		require.Empty(t, e.LastCommit())

		_, err := Read(def, wrapper, repo, resolver, create.Id())
		require.True(t, entity.IsErrNotFound(err))
	})
}

// Repair reloads the committed state after the ref moved, so that the pending
// or write-through operations can be committed on top of it.
func TestRepair(t *testing.T) {
	// setup returns a stale copy of an entity, and the up-to-date one
	setup := func(t *testing.T) (repository.ClockedRepo, entity.Resolvers, Definition, *Foo, *Foo, identity.Interface, identity.Interface) {
		repo, id1, id2, resolver, def := makeTestContext()

		e := wrapper(New(def))
		e.Append(newOp1(id1, "foo"))
		require.NoError(t, e.Commit(repo))

		stale, err := Read(def, wrapper, repo, resolver, e.Id())
		require.NoError(t, err)

		e.Append(newOp2(id1, "moved"))
		require.NoError(t, e.Commit(repo))

		return repo, resolver, def, stale, e, id1, id2
	}

	requireUnchanged := func(t *testing.T, e *Foo, f func()) {
		t.Helper()
		ops, lastCommit := e.Operations(), e.LastCommit()
		createTime, editTime := e.CreateLamportTime(), e.EditLamportTime()
		f()
		require.Equal(t, ops, e.Operations())
		require.Equal(t, lastCommit, e.LastCommit())
		require.Equal(t, createTime, e.CreateLamportTime())
		require.Equal(t, editTime, e.EditLamportTime())
	}

	t.Run("nothing moved", func(t *testing.T) {
		repo, resolver, _, _, moved, _, _ := setup(t)

		counting := &countingRepo{ClockedRepo: repo}
		requireUnchanged(t, moved, func() {
			require.NoError(t, moved.Repair(counting, resolver))
		})
		require.Zero(t, counting.reads)
	})

	t.Run("not found", func(t *testing.T) {
		repo, id1, _, resolver, def := makeTestContext()

		e := wrapper(New(def))
		e.Append(newOp1(id1, "foo"))
		require.True(t, entity.IsErrNotFound(e.Repair(repo, resolver)))
		require.True(t, e.NeedCommit())
	})

	t.Run("pending operations", func(t *testing.T) {
		repo, resolver, def, stale, moved, id1, _ := setup(t)

		op := newOp2(id1, "pending")
		stale.Append(op)
		require.ErrorIs(t, stale.Commit(repo), repository.ErrRefChanged)

		require.NoError(t, stale.Repair(repo, resolver))
		require.Equal(t, moved.LastCommit(), stale.LastCommit())
		require.True(t, stale.NeedCommit())
		require.NoError(t, stale.Commit(repo))

		read, err := Read(def, wrapper, repo, resolver, stale.Id())
		require.NoError(t, err)
		require.Equal(t, append(moved.Operations(), op), read.Operations())
		assertEqualEntities(t, stale.Entity, read.Entity)
	})

	t.Run("write-through operations", func(t *testing.T) {
		repo, resolver, def, stale, moved, id1, _ := setup(t)

		pending := newOp2(id1, "pending")
		stale.Append(pending)

		op := newOp2(id1, "write-through")
		require.ErrorIs(t, stale.CommitOperations(repo, []Operation{op}), repository.ErrRefChanged)

		require.NoError(t, stale.Repair(repo, resolver))
		require.NoError(t, stale.CommitOperations(repo, []Operation{op}))

		// the pending operation is still there
		require.True(t, stale.NeedCommit())
		require.Equal(t, append(moved.Operations(), op, pending), stale.Operations())

		read, err := Read(def, wrapper, repo, resolver, stale.Id())
		require.NoError(t, err)
		require.Equal(t, append(moved.Operations(), op), read.Operations())
	})

	// Only the new commits are read when they descend from the last known one.
	t.Run("incremental", func(t *testing.T) {
		repo, resolver, def, stale, moved, id1, id2 := setup(t)

		// two authors means two commits, plus one, plus the one from setup
		moved.Append(newOp2(id2, "a"))
		moved.Append(newOp2(id1, "b"))
		require.NoError(t, moved.Commit(repo))
		moved.Append(newOp2(id2, "c"))
		require.NoError(t, moved.Commit(repo))

		counting := &countingRepo{ClockedRepo: repo}
		require.NoError(t, stale.Repair(counting, resolver))
		require.Equal(t, 4, counting.reads)

		read, err := Read(def, wrapper, repo, resolver, stale.Id())
		require.NoError(t, err)
		assertEqualEntities(t, read.Entity, stale.Entity)
		assertEqualEntities(t, moved.Entity, stale.Entity)
	})

	// A merge whose branches both descend from the last known commit can still
	// be read incrementally.
	t.Run("incremental merge", func(t *testing.T) {
		repo, resolver, def, stale, moved, id1, id2 := setup(t)

		// a concurrent branch, forked from the stale state
		branch, err := (&operationPack{
			Author:     id2,
			Operations: []Operation{newOp2(id2, "branch")},
			EditTime:   stale.EditLamportTime() + 1,
		}).Write(def, repo, stale.LastCommit())
		require.NoError(t, err)

		merge, err := (&operationPack{
			Author:   id1,
			EditTime: moved.EditLamportTime() + 1,
		}).Write(def, repo, moved.LastCommit(), branch)
		require.NoError(t, err)
		require.NoError(t, repo.UpdateRef(def.Namespace, moved.Id().String(), moved.LastCommit(), merge))

		// the merge, and both branches
		counting := &countingRepo{ClockedRepo: repo}
		require.NoError(t, stale.Repair(counting, resolver))
		require.Equal(t, 3, counting.reads)

		read, err := Read(def, wrapper, repo, resolver, stale.Id())
		require.NoError(t, err)
		require.Len(t, read.Operations(), 3)
		assertEqualEntities(t, read.Entity, stale.Entity)
	})

	// A merge brings operations ordered in the middle of the known ones, which
	// requires to read everything again.
	t.Run("diverged history", func(t *testing.T) {
		repoA, repoB, _, id1, id2, resolver, def := makeTestContextRemote(t)

		eA := wrapper(New(def))
		eA.Append(newOp1(id1, "foo"))
		require.NoError(t, eA.Commit(repoA))
		_, err := Push(def, repoA, "remote")
		require.NoError(t, err)
		require.NoError(t, Pull(def, wrapper, repoB, resolver, "remote", id2))

		eB, err := Read(def, wrapper, repoB, resolver, eA.Id())
		require.NoError(t, err)
		eB.Append(newOp2(id2, "from B"))
		require.NoError(t, eB.Commit(repoB))
		_, err = Push(def, repoB, "remote")
		require.NoError(t, err)

		eA.Append(newOp2(id1, "from A"))
		require.NoError(t, eA.Commit(repoA))

		// merge moves the ref of repoA behind eA's back
		require.NoError(t, Pull(def, wrapper, repoA, resolver, "remote", id1))
		ref, err := repoA.ResolveRef(def.Namespace, eA.Id().String())
		require.NoError(t, err)
		head, err := repoA.ReadCommit(ref)
		require.NoError(t, err)
		require.Len(t, head.Parents, 2)

		require.NoError(t, eA.Repair(repoA, resolver))

		read, err := Read(def, wrapper, repoA, resolver, eA.Id())
		require.NoError(t, err)
		require.Len(t, read.Operations(), 3)
		assertEqualEntities(t, read.Entity, eA.Entity)
	})

	t.Run("invalid new data", func(t *testing.T) {
		// a forged commit is added on top of the moved entity
		for _, tc := range []struct {
			name     string
			editTime func(moved *Foo) lamport.Time
			err      string
		}{
			{"clock not after parent", func(moved *Foo) lamport.Time { return moved.EditLamportTime() }, "lamport clock ordering doesn't match the DAG"},
			{"clock too far", func(moved *Foo) lamport.Time { return moved.EditLamportTime() + 2_000_000 }, "lamport clock jumping too far"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				repo, resolver, _, stale, moved, id1, _ := setup(t)

				opp := &operationPack{
					Author:     id1,
					Operations: []Operation{newOp1(id1, "forged")},
					EditTime:   tc.editTime(moved),
				}
				forged, err := opp.Write(stale.Definition, repo, moved.LastCommit())
				require.NoError(t, err)
				require.NoError(t, repo.UpdateRef(stale.Namespace, stale.Id().String(), moved.LastCommit(), forged))

				requireUnchanged(t, stale, func() {
					require.ErrorContains(t, stale.Repair(repo, resolver), tc.err)
				})
			})
		}
	})

	t.Run("concurrent readers", func(t *testing.T) {
		repo, resolver, _, stale, moved, id1, _ := setup(t)

		var wg sync.WaitGroup
		done := make(chan struct{})
		for range 4 {
			wg.Go(func() {
				for {
					select {
					case <-done:
						return
					default:
					}
					_ = stale.Operations()
					_ = stale.Id()
					_ = stale.LastCommit()
					_ = stale.NeedCommit()
					_ = stale.EditLamportTime()
				}
			})
		}

		for i := range 5 {
			moved.Append(newOp2(id1, fmt.Sprintf("moved %d", i)))
			require.NoError(t, moved.Commit(repo))

			stale.Append(newOp2(id1, fmt.Sprintf("stale %d", i)))
			require.ErrorIs(t, stale.Commit(repo), repository.ErrRefChanged)
			require.NoError(t, stale.Repair(repo, resolver))
			require.NoError(t, stale.Commit(repo))
			require.NoError(t, moved.Repair(repo, resolver))
		}

		close(done)
		wg.Wait()

		assertEqualEntities(t, moved.Entity, stale.Entity)
	})
}

func assertEqualEntities(t *testing.T, a, b *Entity) {
	t.Helper()

	// testify doesn't support comparing functions and systematically fail if they are not nil
	// so we have to set them to nil temporarily

	backOpUnA := a.Definition.OperationUnmarshaler
	backOpUnB := b.Definition.OperationUnmarshaler

	a.Definition.OperationUnmarshaler = nil
	b.Definition.OperationUnmarshaler = nil

	defer func() {
		a.Definition.OperationUnmarshaler = backOpUnA
		b.Definition.OperationUnmarshaler = backOpUnB
	}()

	require.Equal(t, a, b)
}
