package bug

import (
	"github.com/MichaelMure/git-bug/repository"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func createRepo(bare bool) *repository.GitRepo {
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

	return repo
}

func cleanupRepo(repo repository.Repo) error {
	path := repo.GetPath()
	// fmt.Println("Cleaning repo:", path)
	return os.RemoveAll(path)
}

func setupRepos(t testing.TB) (repoA, repoB, remote *repository.GitRepo) {
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

	bug1, err := Create(rene, unix, "bug1", "message")
	checkErr(t, err)
	err = bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote --> B
	_, err = Push(repoA, "origin")
	checkErr(t, err)

	err = Pull(repoB, "origin")
	checkErr(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	bug2, err := Create(rene, unix, "bug2", "message")
	checkErr(t, err)
	err = bug2.Commit(repoB)
	checkErr(t, err)

	_, err = Push(repoB, "origin")
	checkErr(t, err)

	err = Pull(repoA, "origin")
	checkErr(t, err)

	bugs = allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func checkErr(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func allBugs(t testing.TB, bugs <-chan StreamedBug) []*Bug {
	var result []*Bug
	for streamed := range bugs {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Bug)
	}
	return result
}

func TestRebaseTheirs(t *testing.T) {
	_RebaseTheirs(t)
}

func BenchmarkRebaseTheirs(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseTheirs(b)
	}
}

func _RebaseTheirs(t testing.TB) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := Create(rene, unix, "bug1", "message")
	checkErr(t, err)
	err = bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	checkErr(t, err)

	bug2, err := ReadLocalBug(repoB, bug1.Id())
	checkErr(t, err)

	AddComment(bug2, rene, unix, "message2")
	AddComment(bug2, rene, unix, "message3")
	AddComment(bug2, rene, unix, "message4")
	err = bug2.Commit(repoB)
	checkErr(t, err)

	// B --> remote
	_, err = Push(repoB, "origin")
	checkErr(t, err)

	// remote --> A
	err = Pull(repoA, "origin")
	checkErr(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := ReadLocalBug(repoA, bug1.Id())
	checkErr(t, err)

	if nbOps(bug3) != 4 {
		t.Fatal("Unexpected number of operations")
	}
}

func TestRebaseOurs(t *testing.T) {
	_RebaseOurs(t)
}

func BenchmarkRebaseOurs(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseOurs(b)
	}
}

func _RebaseOurs(t testing.TB) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := Create(rene, unix, "bug1", "message")
	checkErr(t, err)
	err = bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	checkErr(t, err)

	AddComment(bug1, rene, unix, "message2")
	AddComment(bug1, rene, unix, "message3")
	AddComment(bug1, rene, unix, "message4")
	err = bug1.Commit(repoA)
	checkErr(t, err)

	AddComment(bug1, rene, unix, "message5")
	AddComment(bug1, rene, unix, "message6")
	AddComment(bug1, rene, unix, "message7")
	err = bug1.Commit(repoA)
	checkErr(t, err)

	AddComment(bug1, rene, unix, "message8")
	AddComment(bug1, rene, unix, "message9")
	AddComment(bug1, rene, unix, "message10")
	err = bug1.Commit(repoA)
	checkErr(t, err)

	// remote --> A
	err = Pull(repoA, "origin")
	checkErr(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug2, err := ReadLocalBug(repoA, bug1.Id())
	checkErr(t, err)

	if nbOps(bug2) != 10 {
		t.Fatal("Unexpected number of operations")
	}
}

func nbOps(b *Bug) int {
	it := NewOperationIterator(b)
	counter := 0
	for it.Next() {
		counter++
	}
	return counter
}

func TestRebaseConflict(t *testing.T) {
	_RebaseConflict(t)
}

func BenchmarkRebaseConflict(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseConflict(b)
	}
}

func _RebaseConflict(t testing.TB) {
	repoA, repoB, remote := setupRepos(t)
	defer cleanupRepos(repoA, repoB, remote)

	bug1, err := Create(rene, unix, "bug1", "message")
	checkErr(t, err)
	err = bug1.Commit(repoA)
	checkErr(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	checkErr(t, err)

	AddComment(bug1, rene, unix, "message2")
	AddComment(bug1, rene, unix, "message3")
	AddComment(bug1, rene, unix, "message4")
	err = bug1.Commit(repoA)
	checkErr(t, err)

	AddComment(bug1, rene, unix, "message5")
	AddComment(bug1, rene, unix, "message6")
	AddComment(bug1, rene, unix, "message7")
	err = bug1.Commit(repoA)
	checkErr(t, err)

	AddComment(bug1, rene, unix, "message8")
	AddComment(bug1, rene, unix, "message9")
	AddComment(bug1, rene, unix, "message10")
	err = bug1.Commit(repoA)
	checkErr(t, err)

	bug2, err := ReadLocalBug(repoB, bug1.Id())
	checkErr(t, err)

	AddComment(bug2, rene, unix, "message11")
	AddComment(bug2, rene, unix, "message12")
	AddComment(bug2, rene, unix, "message13")
	err = bug2.Commit(repoB)
	checkErr(t, err)

	AddComment(bug2, rene, unix, "message14")
	AddComment(bug2, rene, unix, "message15")
	AddComment(bug2, rene, unix, "message16")
	err = bug2.Commit(repoB)
	checkErr(t, err)

	AddComment(bug2, rene, unix, "message17")
	AddComment(bug2, rene, unix, "message18")
	AddComment(bug2, rene, unix, "message19")
	err = bug2.Commit(repoB)
	checkErr(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	checkErr(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	checkErr(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := ReadLocalBug(repoB, bug1.Id())
	checkErr(t, err)

	if nbOps(bug3) != 19 {
		t.Fatal("Unexpected number of operations")
	}

	// B --> remote
	_, err = Push(repoB, "origin")
	checkErr(t, err)

	// remote --> A
	err = Pull(repoA, "origin")
	checkErr(t, err)

	bugs = allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug4, err := ReadLocalBug(repoA, bug1.Id())
	checkErr(t, err)

	if nbOps(bug4) != 19 {
		t.Fatal("Unexpected number of operations")
	}
}
