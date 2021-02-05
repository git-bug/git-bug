package dag

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

func allEntities(t testing.TB, bugs <-chan StreamedEntity) []*Entity {
	t.Helper()

	var result []*Entity
	for streamed := range bugs {
		require.NoError(t, streamed.Err)

		result = append(result, streamed.Entity)
	}
	return result
}

func TestPushPull(t *testing.T) {
	repoA, repoB, remote, id1, id2, def := makeTestContextRemote(t)
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	// A --> remote --> B
	e := New(def)
	e.Append(newOp1(id1, "foo"))

	err := e.Commit(repoA)
	require.NoError(t, err)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	err = Pull(def, repoB, "remote")
	require.NoError(t, err)

	entities := allEntities(t, ReadAll(def, repoB))
	require.Len(t, entities, 1)

	// B --> remote --> A
	e = New(def)
	e.Append(newOp2(id2, "bar"))

	err = e.Commit(repoB)
	require.NoError(t, err)

	_, err = Push(def, repoB, "remote")
	require.NoError(t, err)

	err = Pull(def, repoA, "remote")
	require.NoError(t, err)

	entities = allEntities(t, ReadAll(def, repoB))
	require.Len(t, entities, 2)
}

func TestListLocalIds(t *testing.T) {
	repoA, repoB, remote, id1, id2, def := makeTestContextRemote(t)
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	// A --> remote --> B
	e := New(def)
	e.Append(newOp1(id1, "foo"))
	err := e.Commit(repoA)
	require.NoError(t, err)

	e = New(def)
	e.Append(newOp2(id2, "bar"))
	err = e.Commit(repoA)
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 0)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	_, err = Fetch(def, repoB, "remote")
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 0)

	err = Pull(def, repoB, "remote")
	require.NoError(t, err)

	listLocalIds(t, def, repoA, 2)
	listLocalIds(t, def, repoB, 2)
}

func listLocalIds(t *testing.T, def Definition, repo repository.RepoData, expectedCount int) {
	ids, err := ListLocalIds(def, repo)
	require.NoError(t, err)
	require.Len(t, ids, expectedCount)
}

func assertMergeResults(t *testing.T, expected []entity.MergeResult, results <-chan entity.MergeResult) {
	t.Helper()

	var allResults []entity.MergeResult
	for result := range results {
		allResults = append(allResults, result)
	}

	require.Equal(t, len(expected), len(allResults))

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Id < allResults[j].Id
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Id < expected[j].Id
	})

	for i, result := range allResults {
		require.NoError(t, result.Err)

		require.Equal(t, expected[i].Id, result.Id)
		require.Equal(t, expected[i].Status, result.Status)

		switch result.Status {
		case entity.MergeStatusNew, entity.MergeStatusUpdated:
			require.NotNil(t, result.Entity)
			require.Equal(t, expected[i].Id, result.Entity.Id())
		}

		i++
	}
}

func TestMerge(t *testing.T) {
	repoA, repoB, remote, id1, id2, def := makeTestContextRemote(t)
	defer repository.CleanupTestRepos(repoA, repoB, remote)

	// SCENARIO 1
	// if the remote Entity doesn't exist locally, it's created

	// 2 entities in repoA + push to remote
	e1 := New(def)
	e1.Append(newOp1(id1, "foo"))
	err := e1.Commit(repoA)
	require.NoError(t, err)

	e2 := New(def)
	e2.Append(newOp2(id2, "bar"))
	err = e2.Commit(repoA)
	require.NoError(t, err)

	_, err = Push(def, repoA, "remote")
	require.NoError(t, err)

	// repoB: fetch + merge from remote

	_, err = Fetch(def, repoB, "remote")
	require.NoError(t, err)

	results := MergeAll(def, repoB, "remote")

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1.Id(),
			Status: entity.MergeStatusNew,
		},
		{
			Id:     e2.Id(),
			Status: entity.MergeStatusNew,
		},
	}, results)

	// SCENARIO 2
	// if the remote and local Entity have the same state, nothing is changed

	results = MergeAll(def, repoB, "remote")

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1.Id(),
			Status: entity.MergeStatusNothing,
		},
		{
			Id:     e2.Id(),
			Status: entity.MergeStatusNothing,
		},
	}, results)

	// SCENARIO 3
	// if the local Entity has new commits but the remote don't, nothing is changed

	e1.Append(newOp1(id1, "barbar"))
	err = e1.Commit(repoA)
	require.NoError(t, err)

	e2.Append(newOp2(id2, "barbarbar"))
	err = e2.Commit(repoA)
	require.NoError(t, err)

	results = MergeAll(def, repoA, "remote")

	assertMergeResults(t, []entity.MergeResult{
		{
			Id:     e1.Id(),
			Status: entity.MergeStatusNothing,
		},
		{
			Id:     e2.Id(),
			Status: entity.MergeStatusNothing,
		},
	}, results)

	// SCENARIO 4
	// if the remote has new commit, the local bug is updated to match the same history
	// (fast-forward update)
}
