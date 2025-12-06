package repository

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoGitRepo(t *testing.T) {
	// Plain
	plainRepo := CreateGoGitTestRepo(t, false)
	plainRoot := goGitRepoDir(t, plainRepo)
	require.NoError(t, plainRepo.Close())
	plainGitDir := filepath.Join(plainRoot, ".git")

	// Bare
	bareRepo := CreateGoGitTestRepo(t, true)
	bareRoot := goGitRepoDir(t, bareRepo)
	require.NoError(t, bareRepo.Close())
	bareGitDir := bareRoot

	tests := []struct {
		inPath  string
		outPath string
		err     bool
	}{
		// errors
		{"/", "", true},
		// parent dir of a repo
		{filepath.Dir(plainRoot), "", true},

		// Plain repo
		{plainRoot, plainGitDir, false},
		{plainGitDir, plainGitDir, false},
		{path.Join(plainGitDir, "objects"), plainGitDir, false},

		// Bare repo
		{bareRoot, bareGitDir, false},
		{bareGitDir, bareGitDir, false},
		{path.Join(bareGitDir, "objects"), bareGitDir, false},
	}

	for i, tc := range tests {
		r, err := OpenGoGitRepo(tc.inPath, namespace, nil)

		if tc.err {
			require.Error(t, err, i)
		} else {
			require.NoError(t, err, i)
			assert.Equal(t, filepath.ToSlash(tc.outPath), filepath.ToSlash(r.path), i)
			require.NoError(t, r.Close())
		}
	}
}

func TestGoGitRepo(t *testing.T) {
	RepoTest(t, CreateGoGitTestRepo)
}

func TestGoGitRepo_Head(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	// a new repository's HEAD points to refs/heads/master, which doesn't exist yet
	_, err := repo.Head()
	require.ErrorIs(t, err, ErrNotFound)

	blobHash, err := repo.StoreData(randomData())
	require.NoError(t, err)
	treeHash, err := repo.StoreTree([]TreeEntry{{ObjectType: Blob, Hash: blobHash, Name: "blob"}})
	require.NoError(t, err)
	commit, err := repo.StoreCommit(treeHash)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateRef("refs/heads/master", "", commit))

	meta, err := repo.Head()
	require.NoError(t, err)
	require.Equal(t, RefMeta{
		Name:      "refs/heads/master",
		ShortName: "master",
		Type:      GitRefTypeBranch,
		Hash:      string(commit),
	}, meta)
}

func TestGoGitRepo_ConcurrentUpdateRef(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)

	commits := make([]Hash, 21)
	for i := range commits {
		blobHash, err := repo.StoreData(randomData())
		require.NoError(t, err)
		treeHash, err := repo.StoreTree([]TreeEntry{{ObjectType: Blob, Hash: blobHash, Name: "blob"}})
		require.NoError(t, err)
		commits[i], err = repo.StoreCommit(treeHash)
		require.NoError(t, err)
	}

	ref := "refs/concurrent/update"
	require.NoError(t, repo.UpdateRef(ref, "", commits[0]))

	// every writer moves the ref from the same commit to its own: exactly one must succeed
	candidates := commits[1:]
	errs := make([]error, len(candidates))
	var wg sync.WaitGroup
	for i, commit := range candidates {
		wg.Go(func() { errs[i] = repo.UpdateRef(ref, commits[0], commit) })
	}
	wg.Wait()

	var winners []Hash
	for i, err := range errs {
		if err == nil {
			winners = append(winners, candidates[i])
			continue
		}
		require.ErrorIs(t, err, ErrRefChanged)
	}
	require.Len(t, winners, 1)

	h, err := repo.ResolveRef(ref)
	require.NoError(t, err)
	require.Equal(t, winners[0], h)
}

func TestGoGitRepo_Indexes(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	plainRoot := goGitRepoDir(t, repo)

	// Can create indices
	indexA, err := repo.GetIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "index_meta.json"))
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "store", "root.bolt"))

	indexB, err := repo.GetIndex("b")
	require.NoError(t, err)
	require.NotZero(t, indexB)
	require.DirExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "b"))

	// Can get an existing index
	indexA, err = repo.GetIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)
}

func TestGoGit_DetectsSubmodules(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	expected := filepath.Join(goGitRepoDir(t, repo), "/.git")

	d := t.TempDir()
	rel, err := filepath.Rel(d, expected)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, ".git"), []byte(fmt.Sprintf("gitdir: %s", rel)), 0600)
	require.NoError(t, err)

	result, err := detectGitPath(d, 0)
	assert.Empty(t, err)
	assert.Equal(t, expected, result)
}
