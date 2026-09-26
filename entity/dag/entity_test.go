package dag

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
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
