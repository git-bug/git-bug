package tests

import (
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/misc/random_bugs"
	"github.com/MichaelMure/git-bug/repository"
)

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
