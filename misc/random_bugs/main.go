// +build ignore

package main

import (
	"os"

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

	repo, err := repository.NewGitRepo(dir, func(repo *repository.GitRepo) error {
		return nil
	})
	if err != nil {
		panic(err)
	}

	rb.GenerateRandomBugs(repo, rb.DefaultOptions())
}
