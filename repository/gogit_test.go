package repository

import (
	"os"
	"path"
	"path/filepath"
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

func TestGoGitRepo_Indexes(t *testing.T) {
	repo := CreateGoGitTestRepo(t, false)
	plainRoot := goGitRepoDir(t, repo)

	// Can create indices
	indexA, err := repo.GetIndex("a")
	require.NoError(t, err)
	require.NotZero(t, indexA)
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "index_meta.json"))
	require.FileExists(t, filepath.Join(plainRoot, ".git", namespace, "indexes", "a", "store"))

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
	expectedPath := "../foo/bar"
	submoduleData := "gitdir: " + expectedPath
	d := t.TempDir()
	if f, err := os.Create(filepath.Join(d, ".git")); err != nil {
		t.Fatal("could not create necessary temp file:", err)
	} else {
		t.Log(f.Name())
		if _, err := f.Write([]byte(submoduleData)); err != nil {
			t.Fatal("could not write necessary data to temp file:", err)
		}
		_ = f.Close()
	}
	result, err := detectGitPath(d)
	assert.Empty(t, err)
	assert.Equal(t, expectedPath, result)
}
