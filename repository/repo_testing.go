package repository

import (
	"log"
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
	t.Run("Read/Write data", func(t *testing.T) {
		repo := creator(false)
		defer cleaner(repo)

		data := []byte("hello")

		h, err := repo.StoreData(data)
		require.NoError(t, err)
		assert.NotEmpty(t, h)

		read, err := repo.ReadData(h)
		require.NoError(t, err)
		assert.Equal(t, data, read)
	})

	t.Run("Local config", func(t *testing.T) {
		repo := creator(false)
		defer cleaner(repo)

		testConfig(t, repo.LocalConfig())
	})
}
