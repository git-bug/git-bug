// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	repo := CreateTestRepo(false)
	defer CleanupTestRepos(t, repo)

	err := repo.LocalConfig().StoreString("section.key", "value")
	assert.NoError(t, err)

	val, err := repo.LocalConfig().ReadString("section.key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	err = repo.LocalConfig().StoreString("section.true", "true")
	assert.NoError(t, err)

	val2, err := repo.LocalConfig().ReadBool("section.true")
	assert.NoError(t, err)
	assert.Equal(t, true, val2)

	configs, err := repo.LocalConfig().ReadAll("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key":  "value",
		"section.true": "true",
	})

	err = repo.LocalConfig().RemoveAll("section.true")
	assert.NoError(t, err)

	configs, err = repo.LocalConfig().ReadAll("section")
	assert.NoError(t, err)
	assert.Equal(t, configs, map[string]string{
		"section.key": "value",
	})

	_, err = repo.LocalConfig().ReadBool("section.true")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = repo.LocalConfig().RemoveAll("section.nonexistingkey")
	assert.Error(t, err)

	err = repo.LocalConfig().RemoveAll("section.key")
	assert.NoError(t, err)

	_, err = repo.LocalConfig().ReadString("section.key")
	assert.Equal(t, ErrNoConfigEntry, err)

	err = repo.LocalConfig().RemoveAll("nonexistingsection")
	assert.Error(t, err)

	err = repo.LocalConfig().RemoveAll("section")
	assert.Error(t, err)

	_, err = repo.LocalConfig().ReadString("section.key")
	assert.Error(t, err)

	err = repo.LocalConfig().RemoveAll("section.key")
	assert.Error(t, err)
}

// checkStoreCommit creates a commit and checks if it has been signed.
// See https://git-scm.com/docs/git-log#Documentation/git-log.txt-emGem
// for possible signature status values.
func checkStoreCommit(t *testing.T, repo *GitRepo, expectedSignedStatus string) {
	content := fmt.Sprintf("file content %d", repo.CreateTime())
	blobHash, err := repo.StoreData([]byte(content))
	assert.NoError(t, err)

	var entries = []TreeEntry{{Blob, blobHash, "filename"}}
	treeHash, err := repo.StoreTree(entries)
	assert.NoError(t, err)
	commitHash, err := repo.StoreCommit(treeHash)
	assert.NoError(t, err)

	signedStatus, err := repo.runGitCommand("log", "--pretty=%G?", commitHash.String())
	assert.NoError(t, err)
	assert.Equal(t, expectedSignedStatus, signedStatus)
}

func TestGitRepo_StoreCommit(t *testing.T) {
	repo := CreateTestRepo(false)
	defer CleanupTestRepos(t, repo)

	// Commit and expect no signature.
	checkStoreCommit(t,repo, "N")

	// Commit and expect a good signature with unknown validity.
	SetupSigningKey(t, repo, "a@e.org")
	checkStoreCommit(t, repo, "U")
}
