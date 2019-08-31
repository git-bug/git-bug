package repository

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

// This is intended for testing only

func CreateTestRepo(bare bool) *GitRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("Creating repo:", dir)

	var creator func(string) (*GitRepo, error)

	if bare {
		creator = InitBareGitRepo
	} else {
		creator = InitGitRepo
	}

	repo, err := creator(dir)
	if err != nil {
		log.Fatal(err)
	}

	if err := repo.StoreConfig("user.name", "testuser"); err != nil {
		log.Fatal("failed to set user.name for test repository: ", err)
	}
	if err := repo.StoreConfig("user.email", "testuser@example.com"); err != nil {
		log.Fatal("failed to set user.email for test repository: ", err)
	}

	return repo
}

func CleanupTestRepos(t testing.TB, repos ...Repo) {
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
		t.Fatal(firstErr)
	}
}

func SetupReposAndRemote(t testing.TB) (repoA, repoB, remote *GitRepo) {
	repoA = CreateTestRepo(false)
	repoB = CreateTestRepo(false)
	remote = CreateTestRepo(true)

	remoteAddr := "file://" + remote.GetPath()

	err := repoA.AddRemote("origin", remoteAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = repoB.AddRemote("origin", remoteAddr)
	if err != nil {
		t.Fatal(err)
	}

	return repoA, repoB, remote
}
