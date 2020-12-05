package main

import (
	"os"

	"github.com/MichaelMure/git-bug/bug"
	rb "github.com/MichaelMure/git-bug/misc/random_bugs"
	"github.com/MichaelMure/git-bug/repository"
)

// This program will randomly generate a collection of bugs in the repository
// of the current path
func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	loaders := []repository.ClockLoader{
		bug.ClockLoader,
	}

	repo, err := repository.OpenGoGitRepo(dir, loaders)
	if err != nil {
		panic(err)
	}

	rb.CommitRandomBugs(repo, rb.DefaultOptions())
}
