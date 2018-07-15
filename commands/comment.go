package commands

import (
	"errors"
	"flag"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/commands/input"
	"github.com/MichaelMure/git-bug/repository"
)

var commentFlagSet = flag.NewFlagSet("comment", flag.ExitOnError)

var (
	commentMessageFile = commentFlagSet.String("F", "", "Take the message from the given file. Use - to read the message from the standard input")
	commentMessage     = commentFlagSet.String("m", "", "Provide the new message from the command line")
)

func runComment(repo repository.Repo, args []string) error {
	commentFlagSet.Parse(args)
	args = commentFlagSet.Args()

	var err error

	if len(args) == 0 {
		return errors.New("No bug id provided")
	}
	if len(args) > 1 {
		return errors.New("Only accepting one bug id is supported")
	}

	prefix := args[0]

	if *commentMessageFile != "" && *commentMessage == "" {
		*commentMessage, err = input.FromFile(*commentMessageFile)
		if err != nil {
			return err
		}
	}
	if *commentMessageFile == "" && *commentMessage == "" {
		*commentMessage, err = input.LaunchEditor(repo, messageFilename)
		if err != nil {
			return err
		}
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	bug, err := bug.FindBug(repo, prefix)
	if err != nil {
		return err
	}

	addCommentOp := operations.NewAddCommentOp(author, *commentMessage)

	bug.Append(addCommentOp)

	err = bug.Commit(repo)

	return err
}

var commentCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s comment <id> [<option>...]\n\nOptions:\n", arg0)
		commentFlagSet.PrintDefaults()
	},
	RunMethod: runComment,
}
