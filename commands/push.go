package commands

import (
	"errors"
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
	Description: "Push bugs update to a git remote",
	Usage:       "[<remote>]",
	RunMethod:   runPush,
}
