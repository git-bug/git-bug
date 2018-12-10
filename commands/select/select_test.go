package _select

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

func TestSelect(t *testing.T) {
	repo, err := cache.NewRepoCache(createRepo())
	checkErr(t, err)

	_, _, err = ResolveBug(repo, []string{})
	if err != ErrNoValidId {
		t.Fatal("expected no valid id error, got", err)
	}

	err = Select(repo, "invalid")
	checkErr(t, err)

	_, _, err = ResolveBug(repo, []string{})
	if err == nil {
		t.Fatal("expected invalid bug error")
	}

	// generate a bunch of bugs
	for i := 0; i < 10; i++ {
		_, err := repo.NewBug("title", "message")
		checkErr(t, err)
	}

	// two more for testing
	b1, err := repo.NewBug("title", "message")
	checkErr(t, err)
	b2, err := repo.NewBug("title", "message")
	checkErr(t, err)

	err = Select(repo, b1.Id())
	checkErr(t, err)

	// normal select without args
	b3, _, err := ResolveBug(repo, []string{})
	checkErr(t, err)
	if b3.Id() != b1.Id() {
		t.Fatal("incorrect bug returned")
	}

	// override selection with same id
	b4, _, err := ResolveBug(repo, []string{b1.Id()})
	checkErr(t, err)
	if b4.Id() != b1.Id() {
		t.Fatal("incorrect bug returned")
	}

	// override selection with a prefix
	b5, _, err := ResolveBug(repo, []string{b1.HumanId()})
	checkErr(t, err)
	if b5.Id() != b1.Id() {
		t.Fatal("incorrect bug returned")
	}

	// args that shouldn't override
	b6, _, err := ResolveBug(repo, []string{"arg"})
	checkErr(t, err)
	if b6.Id() != b1.Id() {
		t.Fatal("incorrect bug returned")
	}

	// override with a different id
	b7, _, err := ResolveBug(repo, []string{b2.Id()})
	checkErr(t, err)
	if b7.Id() != b2.Id() {
		t.Fatal("incorrect bug returned")
	}

	err = Clear(repo)
	checkErr(t, err)

	_, _, err = ResolveBug(repo, []string{})
	if err == nil {
		t.Fatal("expected invalid bug error")
	}
}

func createRepo() *repository.GitRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	repo, err := repository.InitGitRepo(dir)
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

func checkErr(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
