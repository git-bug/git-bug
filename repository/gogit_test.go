package repository

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoGitRepo(t *testing.T) {
	// Plain
	plainRoot, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	plainFs := osfs.New(plainRoot)
	// FIXME defer plainFs.RemoveAll(plainRoot)
	// defer (*osfs.OS).RemoveAll(plainFs, plainRoot)

	_, err = InitGoGitRepo(plainRoot, plainFs)
	require.NoError(t, err)
	plainGitDir := path.Join(plainRoot, ".git")

	// Bare
	bareRoot, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	bareFs := osfs.New(bareRoot)
	// FIXME defer bareFs.RemoveAll(bareRoot)

	_, err = InitBareGoGitRepo(bareRoot, bareFs)
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
		{plainGitDir, plainGitDir, false}, //<<<FAILING
		{path.Join(plainGitDir, "objects"), plainGitDir, false},

		// Bare repo
		{bareRoot, bareGitDir, false},
		{bareGitDir, bareGitDir, false},
		{path.Join(bareGitDir, "objects"), bareGitDir, false},
	}

	for i, tc := range tests {
		fs := osfs.New("/")
		r, err := NewGoGitRepo(tc.inPath, nil, fs)

		if tc.err {
			require.Error(t, err, i)
		} else {
			require.NoError(t, err, i)
			assert.Equal(t, filepath.ToSlash(tc.outPath), filepath.ToSlash(r.GitDirPath()), i)
		}
	}
}

func TestGoGitRepo(t *testing.T) {
	RepoTest(t, CreateGoGitTestRepo, CleanupTestRepos)
}
