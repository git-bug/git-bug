package tests

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
)

func createRepo(bare bool) *repository.GitRepo {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Creating repo:", dir)

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

	return repo
}

func cleanupRepo(repo repository.Repo) error {
	path := repo.GetPath()
	fmt.Println("Cleaning repo:", path)
	return os.RemoveAll(path)
}

func setupRepos(t *testing.T) (repoA, repoB, remote *repository.GitRepo) {
	repoA = createRepo(false)
	repoB = createRepo(false)
	remote = createRepo(true)

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

func cleanupRepos(repoA, repoB, remote *repository.GitRepo) {
	cleanupRepo(repoA)
	cleanupRepo(repoB)
	cleanupRepo(remote)
}

func TestPushPull(t *testing.T) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := operations.Create(rene, "bug1", "message")
	checkErr(t, err)
	bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote --> B
	_, err = bug.Push(repoA, "origin")
	checkErr(t, err)

	err = bug.Pull(repoB, os.Stdout, "origin")
	checkErr(t, err)

	bugs := allBugs(t, bug.ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	bug2, err := operations.Create(rene, "bug2", "message")
	checkErr(t, err)
	bug2.Commit(repoB)
	checkErr(t, err)

	_, err = bug.Push(repoB, "origin")
	checkErr(t, err)

	err = bug.Pull(repoA, os.Stdout, "origin")
	checkErr(t, err)

	bugs = allBugs(t, bug.ReadAllLocalBugs(repoA))

	if len(bugs) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func allBugs(t *testing.T, bugs <-chan bug.StreamedBug) []*bug.Bug {
	var result []*bug.Bug
	for streamed := range bugs {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Bug)
	}
	return result
}

func TestRebaseTheirs(t *testing.T) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := operations.Create(rene, "bug1", "message")
	checkErr(t, err)
	bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote
	_, err = bug.Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = bug.Pull(repoB, os.Stdout, "origin")
	checkErr(t, err)

	bug2, err := bug.ReadLocalBug(repoB, bug1.Id())
	checkErr(t, err)

	operations.Comment(bug2, rene, "message2")
	operations.Comment(bug2, rene, "message3")
	operations.Comment(bug2, rene, "message4")
	bug2.Commit(repoB)
	checkErr(t, err)

	// B --> remote
	_, err = bug.Push(repoB, "origin")
	checkErr(t, err)

	// remote --> A
	err = bug.Pull(repoA, os.Stdout, "origin")
	checkErr(t, err)

	bugs := allBugs(t, bug.ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := bug.ReadLocalBug(repoA, bug1.Id())
	checkErr(t, err)

	if nbOps(bug3) != 4 {
		t.Fatal("Unexpected number of operations")
	}
}

func TestRebaseOurs(t *testing.T) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := operations.Create(rene, "bug1", "message")
	checkErr(t, err)
	bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote
	_, err = bug.Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = bug.Pull(repoB, os.Stdout, "origin")
	checkErr(t, err)

	operations.Comment(bug1, rene, "message2")
	operations.Comment(bug1, rene, "message3")
	operations.Comment(bug1, rene, "message4")
	bug1.Commit(repoA)
	checkErr(t, err)

	operations.Comment(bug1, rene, "message5")
	operations.Comment(bug1, rene, "message6")
	operations.Comment(bug1, rene, "message7")
	bug1.Commit(repoA)
	checkErr(t, err)

	operations.Comment(bug1, rene, "message8")
	operations.Comment(bug1, rene, "message9")
	operations.Comment(bug1, rene, "message10")
	bug1.Commit(repoA)
	checkErr(t, err)

	// remote --> A
	err = bug.Pull(repoA, os.Stdout, "origin")
	checkErr(t, err)

	bugs := allBugs(t, bug.ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug2, err := bug.ReadLocalBug(repoA, bug1.Id())
	checkErr(t, err)

	if nbOps(bug2) != 10 {
		t.Fatal("Unexpected number of operations")
	}
}

func nbOps(b *bug.Bug) int {
	it := bug.NewOperationIterator(b)
	counter := 0
	for it.Next() {
		counter++
	}
	return counter
}

func TestRebaseConflict(t *testing.T) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := operations.Create(rene, "bug1", "message")
	checkErr(t, err)
	bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote
	_, err = bug.Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = bug.Pull(repoB, os.Stdout, "origin")
	checkErr(t, err)

	operations.Comment(bug1, rene, "message2")
	operations.Comment(bug1, rene, "message3")
	operations.Comment(bug1, rene, "message4")
	bug1.Commit(repoA)
	checkErr(t, err)

	operations.Comment(bug1, rene, "message5")
	operations.Comment(bug1, rene, "message6")
	operations.Comment(bug1, rene, "message7")
	bug1.Commit(repoA)
	checkErr(t, err)

	operations.Comment(bug1, rene, "message8")
	operations.Comment(bug1, rene, "message9")
	operations.Comment(bug1, rene, "message10")
	bug1.Commit(repoA)
	checkErr(t, err)

	bug2, err := bug.ReadLocalBug(repoB, bug1.Id())
	checkErr(t, err)

	operations.Comment(bug2, rene, "message11")
	operations.Comment(bug2, rene, "message12")
	operations.Comment(bug2, rene, "message13")
	bug2.Commit(repoB)
	checkErr(t, err)

	operations.Comment(bug2, rene, "message14")
	operations.Comment(bug2, rene, "message15")
	operations.Comment(bug2, rene, "message16")
	bug2.Commit(repoB)
	checkErr(t, err)

	operations.Comment(bug2, rene, "message17")
	operations.Comment(bug2, rene, "message18")
	operations.Comment(bug2, rene, "message19")
	bug2.Commit(repoB)
	checkErr(t, err)

	// A --> remote
	_, err = bug.Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = bug.Pull(repoB, os.Stdout, "origin")
	checkErr(t, err)

	bugs := allBugs(t, bug.ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := bug.ReadLocalBug(repoB, bug1.Id())
	checkErr(t, err)

	if nbOps(bug3) != 19 {
		t.Fatal("Unexpected number of operations")
	}

	// B --> remote
	_, err = bug.Push(repoB, "origin")
	checkErr(t, err)

	// remote --> A
	err = bug.Pull(repoA, os.Stdout, "origin")
	checkErr(t, err)

	bugs = allBugs(t, bug.ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug4, err := bug.ReadLocalBug(repoA, bug1.Id())
	checkErr(t, err)

	if nbOps(bug4) != 19 {
		t.Fatal("Unexpected number of operations")
	}
}
