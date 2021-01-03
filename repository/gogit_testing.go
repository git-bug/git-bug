package repository

import (
	"io/ioutil"
	"log"

	"github.com/99designs/keyring"
)

// This is intended for testing only

func CreateGoGitTestRepo(bare bool) TestedRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	var creator func(string) (*GoGitRepo, error)

	if bare {
		creator = InitBareGoGitRepo
	} else {
		creator = InitGoGitRepo
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

	// make sure we use a mock keyring for testing to not interact with the global system
	return &replaceKeyring{
		TestedRepo: repo,
		keyring:    keyring.NewArrayKeyring(nil),
	}
}

func SetupGoGitReposAndRemote() (repoA, repoB, remote TestedRepo) {
	repoA = CreateGoGitTestRepo(false)
	repoB = CreateGoGitTestRepo(false)
	remote = CreateGoGitTestRepo(true)

	err := repoA.AddRemote("origin", remote.GetLocalRemote())
	if err != nil {
		log.Fatal(err)
	}

	err = repoB.AddRemote("origin", remote.GetLocalRemote())
	if err != nil {
		log.Fatal(err)
	}

	return repoA, repoB, remote
}
