package repository

import (
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func CleanupTestRepos(repos ...Repo) {
	var firstErr error
	for _, repo := range repos {
		path := repo.GetPath()
		if strings.HasSuffix(path, "/.git") {
			// for a normal repository (not --bare), we want to remove everything
			// including the parent directory where files are checked out
			path = strings.TrimSuffix(path, "/.git")

			// Testing non-bare repo should also check path is
			// only .git (i.e. ./.git), but doing so, we should
			// try to remove the current directory and hav some
			// trouble. In the present case, this case should not
			// occur.
			// TODO consider warning or error when path == ".git"
		}
		// fmt.Println("Cleaning repo:", path)
		err := os.RemoveAll(path)
		if err != nil {
			log.Println(err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if firstErr != nil {
		log.Fatal(firstErr)
	}
}

type RepoCreator func(bare bool) TestedRepo
type RepoCleaner func(repos ...Repo)

// Test suite for a Repo implementation
func RepoTest(t *testing.T, creator RepoCreator, cleaner RepoCleaner) {
	t.Run("Blob-Tree-Commit-Ref", func(t *testing.T) {
		repo := creator(false)
		defer cleaner(repo)

		// Blob

		data := randomData()

		blobHash1, err := repo.StoreData(data)
		require.NoError(t, err)
		assert.True(t, blobHash1.IsValid())

		blob1Read, err := repo.ReadData(blobHash1)
		require.NoError(t, err)
		assert.Equal(t, data, blob1Read)

		// Tree

		blobHash2, err := repo.StoreData(randomData())
		require.NoError(t, err)
		blobHash3, err := repo.StoreData(randomData())
		require.NoError(t, err)

		tree1 := []TreeEntry{
			{
				ObjectType: Blob,
				Hash:       blobHash1,
				Name:       "blob1",
			},
			{
				ObjectType: Blob,
				Hash:       blobHash2,
				Name:       "blob2",
			},
		}

		treeHash1, err := repo.StoreTree(tree1)
		require.NoError(t, err)
		assert.True(t, treeHash1.IsValid())

		tree1Read, err := repo.ReadTree(treeHash1)
		require.NoError(t, err)
		assert.ElementsMatch(t, tree1, tree1Read)

		tree2 := []TreeEntry{
			{
				ObjectType: Tree,
				Hash:       treeHash1,
				Name:       "tree1",
			},
			{
				ObjectType: Blob,
				Hash:       blobHash3,
				Name:       "blob3",
			},
		}

		treeHash2, err := repo.StoreTree(tree2)
		require.NoError(t, err)
		assert.True(t, treeHash2.IsValid())

		tree2Read, err := repo.ReadTree(treeHash2)
		require.NoError(t, err)
		assert.ElementsMatch(t, tree2, tree2Read)

		// Commit

		commit1, err := repo.StoreCommit(treeHash1)
		require.NoError(t, err)
		assert.True(t, commit1.IsValid())

		treeHash1Read, err := repo.GetTreeHash(commit1)
		require.NoError(t, err)
		assert.Equal(t, treeHash1, treeHash1Read)

		commit2, err := repo.StoreCommitWithParent(treeHash2, commit1)
		require.NoError(t, err)
		assert.True(t, commit2.IsValid())

		treeHash2Read, err := repo.GetTreeHash(commit2)
		require.NoError(t, err)
		assert.Equal(t, treeHash2, treeHash2Read)

		// Ref

		exist1, err := repo.RefExist("refs/bugs/ref1")
		require.NoError(t, err)
		assert.False(t, exist1)

		err = repo.UpdateRef("refs/bugs/ref1", commit2)
		require.NoError(t, err)

		exist1, err = repo.RefExist("refs/bugs/ref1")
		require.NoError(t, err)
		assert.True(t, exist1)

		ls, err := repo.ListRefs("refs/bugs")
		require.NoError(t, err)
		assert.Equal(t, []string{"refs/bugs/ref1"}, ls)

		err = repo.CopyRef("refs/bugs/ref1", "refs/bugs/ref2")
		require.NoError(t, err)

		ls, err = repo.ListRefs("refs/bugs")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"refs/bugs/ref1", "refs/bugs/ref2"}, ls)

		commits, err := repo.ListCommits("refs/bugs/ref2")
		require.NoError(t, err)
		assert.Equal(t, []Hash{commit1, commit2}, commits)

		// Graph

		commit3, err := repo.StoreCommitWithParent(treeHash1, commit1)
		require.NoError(t, err)

		ancestorHash, err := repo.FindCommonAncestor(commit2, commit3)
		require.NoError(t, err)
		assert.Equal(t, commit1, ancestorHash)
	})

	t.Run("Local config", func(t *testing.T) {
		repo := creator(false)
		defer cleaner(repo)

		testConfig(t, repo.LocalConfig())
	})
}

func randomData() []byte {
	var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return b
}
