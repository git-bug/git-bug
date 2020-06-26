// Package repository contains helper methods for working with the Git repo.
package repository

import (
	"testing"
)

func TestGitRepo(t *testing.T) {
	RepoTest(t, CreateTestRepo, CleanupTestRepos)
}
