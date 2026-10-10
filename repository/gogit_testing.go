package repository

import (
	"errors"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"github.com/git-bug/gitconfig"
)

const namespace = "git-bug"

// This is intended for testing only

func CreateGoGitTestRepo(t testing.TB, bare bool) TestedRepo {
	t.Helper()

	dir := t.TempDir()

	var creator func(string, string) (*GoGitRepo, error)

	if bare {
		creator = InitBareGoGitRepo
	} else {
		creator = InitGoGitRepo
	}

	repo, err := creator(dir, namespace)
	if err != nil {
		log.Fatal(err)
	}

	t.Cleanup(func() {
		err := repo.Close()
		if err != nil {
			log.Println(err)
		}
	})

	// don't read or write the configuration of the user or the system
	repo.configEnv = gitconfig.Env{
		GitDir:       repo.path,
		GlobalConfig: []string{filepath.Join(t.TempDir(), ".gitconfig")},
	}

	err = repo.Config().Update(func(f *gitconfig.File) error {
		return errors.Join(
			f.Set("user.name", "testuser"),
			f.Set("user.email", "testuser@example.com"),
		)
	})
	if err != nil {
		log.Fatal("failed to set the user for test repository: ", err)
	}

	// make sure we use a mock keyring for testing to not interact with the global system
	return &replaceKeyring{
		TestedRepo: repo,
		keyring:    keyring.NewArrayKeyring(nil),
	}
}

func SetupGoGitReposAndRemote(t *testing.T) (repoA, repoB, remote TestedRepo) {
	t.Helper()

	repoA = CreateGoGitTestRepo(t, false)
	repoB = CreateGoGitTestRepo(t, false)
	remote = CreateGoGitTestRepo(t, true)

	AddRemote(t, repoA, "origin", remote.GetLocalRemote())
	AddRemote(t, repoB, "origin", remote.GetLocalRemote())

	return repoA, repoB, remote
}

func goGitRepoDir(t *testing.T, repo TestedRepo) string {
	t.Helper()

	dir := repo.GetLocalRemote()
	if strings.HasSuffix(dir, ".git") {
		dir, _ = filepath.Split(dir)
	}

	if dir[len(dir)-1] == filepath.Separator {
		dir = dir[:len(dir)-1]
	}

	return dir
}
