package execenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ***** WARNING ***** - due to the use of testing with an environment
//
//	variable, do NOT use t.Parallel() with these
//	test cases.
func TestGetRepoPath(t *testing.T) {
	t.Run("no Git path provided (use WD)", func(t *testing.T) {
		// WD during tests is the directory containing the test file
		path, err := getRepoPath(&Env{})
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(path, filepath.Join("commands", "execenv")))
	})

	t.Run("flag provided", func(t *testing.T) {
		gitDir, err := os.UserHomeDir()
		require.NoError(t, err)

		path, err := getRepoPath(&Env{
			GitDir: gitDir,
		})
		require.NoError(t, err)
		assert.Equal(t, gitDir, path)
	})

	t.Run("environment variable provided", func(t *testing.T) {
		gitDir, err := os.UserHomeDir()
		require.NoError(t, err)
		t.Setenv("GIT_DIR", gitDir)

		path, err := getRepoPath(&Env{})
		require.NoError(t, err)
		assert.Equal(t, gitDir, path)
	})

	t.Run("GIT_DIR supercedes --git-dir", func(t *testing.T) {
		cacheDir, err := os.UserCacheDir()
		require.NoError(t, err)

		configDir, err := os.UserConfigDir()
		require.NoError(t, err)
		t.Setenv("GIT_DIR", configDir)

		path, err := getRepoPath(&Env{
			GitDir: cacheDir,
		})
		require.NoError(t, err)
		assert.Equal(t, configDir, path)

	})

	t.Run("path does not exist", func(t *testing.T) {
		_, err := getRepoPath(&Env{
			GitDir: filepath.Join("/", "tmp", "does-not-exist"),
		})
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("path is not a directory", func(t *testing.T) {
		gitDir := filepath.Join(os.TempDir(), "plain-file")

		f, err := os.Create(gitDir)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		_, err = getRepoPath(&Env{
			GitDir: gitDir,
		})
		require.ErrorIs(t, err, errNotDirectory)
	})
}
