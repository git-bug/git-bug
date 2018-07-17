package commands

import (
	"errors"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
)

func runCloseBug(repo repository.Repo, args []string) error {
	if len(args) > 1 {
		return errors.New("Only closing one bug at a time is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	prefix := args[0]

	b, err := bug.FindBug(repo, prefix)
	if err != nil {
		return err
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	op := operations.NewSetStatusOp(author, bug.ClosedStatus)

	b.Append(op)

	err = b.Commit(repo)

	return err
}

var closeCmd = &Command{
	Description: "Mark the bug as closed",
	Usage:       "<id>",
	RunMethod:   runCloseBug,
}
