package repository_test

import (
	"testing"

	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestGoGitRepo(t *testing.T) {
	repotest.RepoTest(t, func(t testing.TB, bare bool) repotest.Repo {
		return repotest.NewGoGitRepo(t, bare)
	})
}

func TestMockRepo(t *testing.T) {
	repotest.RepoTest(t, func(t testing.TB, bare bool) repotest.Repo {
		return repository.NewMockRepo()
	})
}
