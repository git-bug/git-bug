package main

import (
	"os"

	"github.com/git-bug/git-bug/entities/board"
	"github.com/git-bug/git-bug/entities/bug"
	rb "github.com/git-bug/git-bug/misc/random_bugs"
	"github.com/git-bug/git-bug/repository"
)

// This program will randomly generate a collection of bugs in the repository
// of the current path
func main() {
	const gitBugNamespace = "git-bug"

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	loaders := []repository.ClockLoader{
		bug.ClockLoader,
		board.ClockLoader,
	}

	repo, err := repository.OpenGoGitRepo(dir, gitBugNamespace, loaders)
	if err != nil {
		panic(err)
	}

	rb.CommitRandomBugs(repo, rb.DefaultOptions())
}
