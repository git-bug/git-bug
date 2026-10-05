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

// git-bug never writes a merge commit with a parent that is also an ancestor of its
// other parent, and such a DAG must be rejected. Reading it must never panic, even
// when crafted so that a commit other than the root passes the root checks.
//
//	root <- X <- A <- merge
//	  ^               |
//	  +---------------+
func TestReadMergeWithAncestorParent(t *testing.T) {
	for _, tc := range []struct {
		name string
		// whether X carries the root operation and a creation time
		craftedX bool
		err      string
	}{
		{"plain", false, "merge commit with a parent that is an ancestor of another"},
		// rejected before reaching the merge
		{"crafted", true, "duplicate operation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, id1, _, resolver, def := makeTestContext()

			rootOp := newOp1(id1, "root")
			e := wrapper(New(def))
			e.Append(rootOp)
			require.NoError(t, e.Commit(repo))
			root := e.LastCommit()

			xPack := &operationPack{
				Author:     id1,
				Operations: []Operation{newOp2(id1, "x")},
				EditTime:   e.EditLamportTime() + 1,
			}
			if tc.craftedX {
				xPack.Operations = []Operation{rootOp}
				xPack.CreateTime = e.CreateLamportTime()
			}
			x, err := xPack.Write(def, repo, root)
			require.NoError(t, err)

			a, err := (&operationPack{
				Author:     id1,
				Operations: []Operation{newOp2(id1, "a")},
				EditTime:   e.EditLamportTime() + 2,
			}).Write(def, repo, x)
			require.NoError(t, err)

			merge, err := (&operationPack{
				Author:   id1,
				EditTime: e.EditLamportTime() + 3,
			}).Write(def, repo, a, root)
			require.NoError(t, err)
			require.NoError(t, repo.UpdateRef(def.Namespace, e.Id().String(), root, merge))

			require.NotPanics(t, func() {
				_, err = Read(def, wrapper, repo, resolver, e.Id())
			})

			require.ErrorContains(t, err, tc.err)
		})
	}
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
		type forge struct {
			ops      []Operation
			editTime lamport.Time
			parents  []repository.Hash
		}
		for _, tc := range []struct {
			name  string
			forge func(stale, moved *Foo, id1 identity.Interface) forge
			err   string
		}{
			{"clock not after parent", func(stale, moved *Foo, id1 identity.Interface) forge {
				return forge{[]Operation{newOp1(id1, "forged")}, moved.EditLamportTime(), []repository.Hash{moved.LastCommit()}}
			}, "lamport clock ordering doesn't match the DAG"},
			{"clock too far", func(stale, moved *Foo, id1 identity.Interface) forge {
				return forge{[]Operation{newOp1(id1, "forged")}, moved.EditLamportTime() + 2_000_000, []repository.Hash{moved.LastCommit()}}
			}, "lamport clock jumping too far"},
			// the known root operation, stored again
			{"duplicate operation", func(stale, moved *Foo, id1 identity.Interface) forge {
				return forge{[]Operation{stale.FirstOp()}, moved.EditLamportTime() + 1, []repository.Hash{moved.LastCommit()}}
			}, "duplicate operation"},
			// the last known commit is an ancestor of the moved one
			{"merge with ancestor parent", func(stale, moved *Foo, id1 identity.Interface) forge {
				return forge{nil, moved.EditLamportTime() + 1, []repository.Hash{moved.LastCommit(), stale.LastCommit()}}
			}, "merge commit with a parent that is an ancestor of another"},
			{"merge with the same parent twice", func(stale, moved *Foo, id1 identity.Interface) forge {
				return forge{nil, moved.EditLamportTime() + 1, []repository.Hash{moved.LastCommit(), moved.LastCommit()}}
			}, "merge commit with a parent that is an ancestor of another"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				repo, resolver, _, stale, moved, id1, _ := setup(t)

				f := tc.forge(stale, moved, id1)
				opp := &operationPack{
					Author:     id1,
					Operations: f.ops,
					EditTime:   f.editTime,
				}
				forged, err := opp.Write(stale.Definition, repo, f.parents...)
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

// BenchmarkRead reads entities with different history shapes, each with about the
// given number of commits.
func BenchmarkRead(b *testing.B) {
	type history struct {
		repo      repository.ClockedRepo
		resolvers entity.Resolvers
		def       Definition
		head      repository.Hash
		e         *Foo
	}

	// build creates an entity, and returns a function writing commits with the
	// given clock on top of the given parents. Merge commits have no operation.
	build := func(b *testing.B) (history, func(editTime lamport.Time, parents ...repository.Hash) repository.Hash) {
		repo, id1, _, resolvers, def := makeTestContext()

		e := wrapper(New(def))
		e.Append(newOp1(id1, "root"))
		require.NoError(b, e.Commit(repo))

		var n int
		write := func(editTime lamport.Time, parents ...repository.Hash) repository.Hash {
			opp := &operationPack{Author: id1, EditTime: editTime}
			if len(parents) == 1 {
				n++
				opp.Operations = []Operation{newOp2(id1, fmt.Sprintf("op %d", n))}
			}
			hash, err := opp.Write(def, repo, parents...)
			require.NoError(b, err)
			return hash
		}

		return history{repo: repo, resolvers: resolvers, def: def, head: e.LastCommit(), e: e}, write
	}

	shapes := []struct {
		name  string
		build func(b *testing.B, commits int) history
	}{
		{"linear", func(b *testing.B, commits int) history {
			h, write := build(b)
			clock := h.e.EditLamportTime()
			for range commits {
				clock++
				h.head = write(clock, h.head)
			}
			return h
		}},
		// Short-lived branches, merged back after two commits on each side: the
		// usual shape of concurrent edition.
		{"merges", func(b *testing.B, commits int) history {
			h, write := build(b)
			clock := h.e.EditLamportTime()
			for range commits / 5 {
				fork := h.head
				branch := fork
				for i := range lamport.Time(2) {
					branch = write(clock+1+i, branch)
					h.head = write(clock+1+i, h.head)
				}
				clock += 3
				h.head = write(clock, h.head, branch)
			}
			return h
		}},
		// A long-lived branch whose clocks lag far behind, merged again and again
		// into a busy trunk: the worst case for the merge-parent check, as its
		// ancestry walks can't stop early.
		{"lagging branch", func(b *testing.B, commits int) history {
			h, write := build(b)
			base := h.e.EditLamportTime()
			branch := h.head
			clock := base + lamport.Time(commits)
			for i := range commits / 5 {
				branch = write(base+1+lamport.Time(i), branch)
				for range 3 {
					clock++
					h.head = write(clock, h.head)
				}
				clock++
				h.head = write(clock, h.head, branch)
			}
			return h
		}},
	}

	for _, shape := range shapes {
		for _, commits := range []int{100, 1000} {
			b.Run(fmt.Sprintf("%s/%d", shape.name, commits), func(b *testing.B) {
				h := shape.build(b, commits)
				// make sure the history is valid
				_, err := read(h.def, wrapper, h.repo, h.resolvers, h.e.Id(), h.head)
				require.NoError(b, err)

				for b.Loop() {
					_, _ = read(h.def, wrapper, h.repo, h.resolvers, h.e.Id(), h.head)
				}
			})
		}
	}
}
