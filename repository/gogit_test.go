package repository

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoGitRepo(t *testing.T) {
	// Plain
	plainRoot, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(plainRoot)

	_, err = InitGoGitRepo(plainRoot, namespace)
	require.NoError(t, err)
	plainGitDir := filepath.Join(plainRoot, ".git")

	// Bare
	bareRoot, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(bareRoot)

	_, err = InitBareGoGitRepo(bareRoot, namespace)
	require.NoError(t, err)
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
		}
	}
}

func TestGoGitRepo(t *testing.T) {
	RepoTest(t, CreateGoGitTestRepo, CleanupTestRepos)
}

func TestGoGitRepo_Indexes(t *testing.T) {
	t.Parallel()

	plainRoot, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	// defer os.RemoveAll(plainRoot)

	repo, err := InitGoGitRepo(plainRoot, namespace)
	require.NoError(t, err)

	// Can create indices
	indexA, err := repo.GetBleveIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "index_meta.json"))
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "store"))

	indexB, err := repo.GetBleveIndex("b")
	require.NoError(t, err)
	require.NotZero(t, indexB)
	require.DirExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "b"))

	// Can get an existing index
	indexA, err = repo.GetBleveIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)

	// Can delete an index
	err = repo.ClearBleveIndex("a")
	require.NoError(t, err)
	require.NoDirExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a"))
}
