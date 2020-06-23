package repository

import (
	"io/ioutil"
	"log"
)

// This is intended for testing only

func CreateTestRepo(bare bool) TestedRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

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

	config := repo.LocalConfig()
	if err := config.StoreString("user.name", "testuser"); err != nil {
		log.Fatal("failed to set user.name for test repository: ", err)
	}
	if err := config.StoreString("user.email", "testuser@example.com"); err != nil {
		log.Fatal("failed to set user.email for test repository: ", err)
	}

	return repo
}

func SetupReposAndRemote() (repoA, repoB, remote TestedRepo) {
	repoA = CreateTestRepo(false)
	repoB = CreateTestRepo(false)
	remote = CreateTestRepo(true)

	remoteAddr := "file://" + remote.GetPath()

	err := repoA.AddRemote("origin", remoteAddr)
	if err != nil {
		log.Fatal(err)
	}

	err = repoB.AddRemote("origin", remoteAddr)
	if err != nil {
		log.Fatal(err)
	}

	return repoA, repoB, remote
}
