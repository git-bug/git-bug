// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitRepo(t *testing.T) {
	RepoTest(t, CreateTestRepo, CleanupTestRepos)
}

// checkStoreCommit creates a commit and checks if it has been signed.
// See https://git-scm.com/docs/git-log#Documentation/git-log.txt-emGem
// for possible signature status values.
func checkStoreCommit(t *testing.T, repo TestedRepo, expectedSignedStatus string) {
	blobHash, err := repo.StoreData([]byte("content"))
	assert.NoError(t, err)

	var entries = []TreeEntry{{Blob, blobHash, "filename"}}
	treeHash, err := repo.StoreTree(entries)
	assert.NoError(t, err)
	commitHash, err := repo.StoreCommit(treeHash)
	assert.NoError(t, err)

	gitRepo := repo.(*GitRepo)
	signedStatus, err := gitRepo.runGitCommand("log", "--pretty=%G?", commitHash.String())
	assert.NoError(t, err)
	assert.Equal(t, expectedSignedStatus, signedStatus)
}

func TestGitRepo_StoreCommit(t *testing.T) {
	repo := CreateTestRepo(false)
	defer CleanupTestRepos(repo)

	// Commit and expect no signature.
	checkStoreCommit(t, repo, "N")

	// Commit and expect a good signature with unknown validity.
	setupSigningKey(t, repo)
	checkStoreCommit(t, repo, "U")
}
