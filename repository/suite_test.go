package repository_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestGoGitRepo(t *testing.T) {
	repotest.RepoTest(t, func(t testing.TB, bare bool) repotest.Repo {
		return repotest.NewGoGitRepo(t, bare)
	})

	repotest.RepoRemoteTest(t, func(t testing.TB, repo *repotest.GoGitRepo) repository.RepoRemote {
		return repo.RepoRemote
	})
}

func TestMockRepo(t *testing.T) {
	repotest.RepoTest(t, func(t testing.TB, bare bool) repotest.Repo {
		return repository.NewMockRepo()
	})
}

func TestGitRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	repotest.RepoRemoteTest(t, func(t testing.TB, repo *repotest.GoGitRepo) repository.RepoRemote {
		remote, err := repository.NewGitRepo(repo.GitDir, repo.RepoConfig)
		require.NoError(t, err)
		return remote
	})
}
