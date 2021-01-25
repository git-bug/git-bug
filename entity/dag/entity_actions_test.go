package dag

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func allEntities(t testing.TB, bugs <-chan StreamedEntity) []*Entity {
	var result []*Entity
	for streamed := range bugs {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Entity)
	}
	return result
}

func TestPushPull(t *testing.T) {
	repoA, repoB, remote, id1, id2, def := makeTestContextRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	// distribute the identities
	_, err := identity.Push(repoA, "origin")
	require.NoError(t, err)
	err = identity.Pull(repoB, "origin")
	require.NoError(t, err)

	// A --> remote --> B
	entity := New(def)
	entity.Append(newOp1(id1, "foo"))

	err = entity.Commit(repoA)
	require.NoError(t, err)

	_, err = Push(def, repoA, "origin")
	require.NoError(t, err)

	err = Pull(def, repoB, "origin")
	require.NoError(t, err)

	entities := allEntities(t, ReadAll(def, repoB))
	require.Len(t, entities, 1)

	// B --> remote --> A
	entity = New(def)
	entity.Append(newOp2(id2, "bar"))

	err = entity.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(def, repoB, "origin")
	require.NoError(t, err)

	err = Pull(def, repoA, "origin")
	require.NoError(t, err)

	entities = allEntities(t, ReadAll(def, repoB))
	require.Len(t, entities, 2)
}

func TestListLocalIds(t *testing.T) {
	repoA, repoB, remote, id1, id2, def := makeTestContextRemote()
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	// distribute the identities
	_, err := identity.Push(repoA, "origin")
	require.NoError(t, err)
	err = identity.Pull(repoB, "origin")
	require.NoError(t, err)

	// A --> remote --> B
	entity := New(def)
	entity.Append(newOp1(id1, "foo"))
	err = entity.Commit(repoA)
	require.NoError(t, err)

	entity = New(def)
	entity.Append(newOp2(id2, "bar"))
	err = entity.Commit(repoA)
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 0)

	_, err = Push(def, repoA, "origin")
	require.NoError(t, err)

	_, err = Fetch(def, repoB, "origin")
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 0)

	err = Pull(def, repoB, "origin")
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 2)
}

func listLocalIds(t *testing.T, def Definition, repo repository.RepoData, expectedCount int) {
	ids, err := ListLocalIds(def, repo)
	require.NoError(t, err)
	require.Len(t, ids, expectedCount)
}
