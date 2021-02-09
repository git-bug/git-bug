package dag

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteRead(t *testing.T) {
	repo, id1, id2, def := makeTestContext()

	entity := New(def)
	require.False(t, entity.NeedCommit())

	entity.Append(newOp1(id1, "foo"))
	entity.Append(newOp2(id1, "bar"))

	require.True(t, entity.NeedCommit())
	require.NoError(t, entity.CommitAdNeeded(repo))
	require.False(t, entity.NeedCommit())

	entity.Append(newOp2(id2, "foobar"))
	require.True(t, entity.NeedCommit())
	require.NoError(t, entity.CommitAdNeeded(repo))
	require.False(t, entity.NeedCommit())

	read, err := Read(def, repo, entity.Id())
	require.NoError(t, err)

	assertEqualEntities(t, entity, read)
}

func assertEqualEntities(t *testing.T, a, b *Entity) {
	// testify doesn't support comparing functions and systematically fail if they are not nil
	// so we have to set them to nil temporarily

	backOpUnA := a.Definition.operationUnmarshaler
	backOpUnB := b.Definition.operationUnmarshaler

	a.Definition.operationUnmarshaler = nil
	b.Definition.operationUnmarshaler = nil

	backIdResA := a.Definition.identityResolver
	backIdResB := b.Definition.identityResolver

	a.Definition.identityResolver = nil
	b.Definition.identityResolver = nil

	defer func() {
		a.Definition.operationUnmarshaler = backOpUnA
		b.Definition.operationUnmarshaler = backOpUnB
		a.Definition.identityResolver = backIdResA
		b.Definition.identityResolver = backIdResB
	}()

	require.Equal(t, a, b)
}
