package test

import (
	"io/ioutil"
	"log"

	"github.com/MichaelMure/git-bug/misc/random_bugs"
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

func CreateFilledRepo(bugNumber int) repository.ClockedRepo {
	repo := CreateRepo(false)

	var seed int64 = 42
	options := random_bugs.DefaultOptions()

	options.BugNumber = bugNumber

	random_bugs.CommitRandomBugsWithSeed(repo, options, seed)
	return repo
}
