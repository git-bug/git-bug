package commands

import (
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
)

func runPush(repo repository.Repo, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	if err := repo.PushRefs(remote, bug.BugsRefPattern+"*"); err != nil {
		return err
	}
	return nil
}

// showCmd defines the "push" subcommand.
var pushCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s push [<remote>]\n", arg0)
	},
	RunMethod: runPush,
}
