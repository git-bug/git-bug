package dag

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteRead(t *testing.T) {
	repo, id1, id2, resolver, def := makeTestContext()

	entity := New(def)
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

	read, err := Read(def, repo, resolver, entity.Id())
	require.NoError(t, err)

	assertEqualEntities(t, entity, read)
}

func TestWriteReadMultipleAuthor(t *testing.T) {
	repo, id1, id2, resolver, def := makeTestContext()

	entity := New(def)

	entity.Append(newOp1(id1, "foo"))
	entity.Append(newOp2(id2, "bar"))

	require.NoError(t, entity.CommitAsNeeded(repo))

	entity.Append(newOp2(id1, "foobar"))
	require.NoError(t, entity.CommitAsNeeded(repo))

	read, err := Read(def, repo, resolver, entity.Id())
	require.NoError(t, err)

	assertEqualEntities(t, entity, read)
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
