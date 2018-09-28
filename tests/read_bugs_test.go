package tests

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/misc/random_bugs"
	"github.com/MichaelMure/git-bug/repository"
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

func createFilledRepo(bugNumber int) repository.ClockedRepo {
	repo := createRepo(false)

	var seed int64 = 42
	options := random_bugs.DefaultOptions()

	options.BugNumber = bugNumber

	random_bugs.CommitRandomBugsWithSeed(repo, options, seed)
	return repo
}

func TestReadBugs(t *testing.T) {
	repo := createFilledRepo(15)
	bugs := bug.ReadAllLocalBugs(repo)
	for b := range bugs {
		if b.Err != nil {
			t.Fatal(b.Err)
		}
	}
}

func benchmarkReadBugs(bugNumber int, t *testing.B) {
	repo := createFilledRepo(bugNumber)
	t.ResetTimer()

	for n := 0; n < t.N; n++ {
		bugs := bug.ReadAllLocalBugs(repo)
		for b := range bugs {
			if b.Err != nil {
				t.Fatal(b.Err)
			}
		}
	}
}

func BenchmarkReadBugs5(b *testing.B)   { benchmarkReadBugs(5, b) }
func BenchmarkReadBugs25(b *testing.B)  { benchmarkReadBugs(25, b) }
func BenchmarkReadBugs150(b *testing.B) { benchmarkReadBugs(150, b) }
