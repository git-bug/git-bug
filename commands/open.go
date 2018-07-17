package commands

import (
	"errors"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
)

func runOpenBug(repo repository.Repo, args []string) error {
	if len(args) > 1 {
		return errors.New("Only opening one bug at a time is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	prefix := args[0]

	b, err := bug.FindBug(repo, prefix)
	if err != nil {
		return err
	}

	op := operations.NewSetStatusOp(bug.OpenStatus)

	b.Append(op)

	err = b.Commit(repo)

	return err
}

var openCmd = &Command{
	Description: "Mark the bug as open",
	Usage:       "<id>",
	RunMethod:   runOpenBug,
}
