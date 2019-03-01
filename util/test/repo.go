package test

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/MichaelMure/git-bug/repository"
)

func CreateRepo(bare bool) *repository.GitRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("Creating repo:", dir)

	var creator func(string) (*repository.GitRepo, error)

	if bare {
		creator = repository.InitBareGitRepo
	} else {
		creator = repository.InitGitRepo
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

func CleanupRepo(repo repository.Repo) error {
	path := repo.GetPath()
	// fmt.Println("Cleaning repo:", path)
	return os.RemoveAll(path)
}

func SetupReposAndRemote(t testing.TB) (repoA, repoB, remote *repository.GitRepo) {
	repoA = CreateRepo(false)
	repoB = CreateRepo(false)
	remote = CreateRepo(true)

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

func CleanupRepos(repoA, repoB, remote *repository.GitRepo) {
	if err := CleanupRepo(repoA); err != nil {
		log.Println(err)
	}
	if err := CleanupRepo(repoB); err != nil {
		log.Println(err)
	}
	if err := CleanupRepo(remote); err != nil {
		log.Println(err)
	}
}
