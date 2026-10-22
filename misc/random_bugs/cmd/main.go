package main

import (
	"context"
	"os"

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

	gitDir, err := repository.FindGitDir(dir)
	if err != nil {
		panic(err)
	}

	repo, err := repository.OpenDefault(gitDir, gitBugNamespace)
	if err != nil {
		panic(err)
	}
	defer repo.Close()

	rb.CommitRandomBugs(context.Background(), repo, rb.DefaultOptions())
}
