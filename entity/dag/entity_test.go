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

// // Merge
//
// merge1 := makeCommit(t, repo)
// merge1 = makeCommit(t, repo, merge1)
// err = repo.UpdateRef("merge1", merge1)
// require.NoError(t, err)
//
// err = repo.UpdateRef("merge2", merge1)
// require.NoError(t, err)
//
// // identical merge
// err = repo.MergeRef("merge1", "merge2")
// require.NoError(t, err)
//
// refMerge1, err := repo.ResolveRef("merge1")
// require.NoError(t, err)
// require.Equal(t, merge1, refMerge1)
// refMerge2, err := repo.ResolveRef("merge2")
// require.NoError(t, err)
// require.Equal(t, merge1, refMerge2)
//
// // fast-forward merge
// merge2 := makeCommit(t, repo, merge1)
// merge2 = makeCommit(t, repo, merge2)
//
// err = repo.UpdateRef("merge2", merge2)
// require.NoError(t, err)
//
// err = repo.MergeRef("merge1", "merge2")
// require.NoError(t, err)
//
// refMerge1, err = repo.ResolveRef("merge1")
// require.NoError(t, err)
// require.Equal(t, merge2, refMerge1)
// refMerge2, err = repo.ResolveRef("merge2")
// require.NoError(t, err)
// require.Equal(t, merge2, refMerge2)
//
// // merge commit
// merge1 = makeCommit(t, repo, merge1)
// err = repo.UpdateRef("merge1", merge1)
// require.NoError(t, err)
//
// merge2 = makeCommit(t, repo, merge2)
// err = repo.UpdateRef("merge2", merge2)
// require.NoError(t, err)
//
// err = repo.MergeRef("merge1", "merge2")
// require.NoError(t, err)
//
// refMerge1, err = repo.ResolveRef("merge1")
// require.NoError(t, err)
// require.NotEqual(t, merge1, refMerge1)
// commitRefMerge1, err := repo.ReadCommit(refMerge1)
// require.NoError(t, err)
// require.ElementsMatch(t, commitRefMerge1.Parents, []Hash{merge1, merge2})
// refMerge2, err = repo.ResolveRef("merge2")
// require.NoError(t, err)
// require.Equal(t, merge2, refMerge2)
