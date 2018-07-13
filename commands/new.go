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

var newFlagSet = flag.NewFlagSet("new", flag.ExitOnError)

var (
	newMessageFile = newFlagSet.String("F", "", "Take the message from the given file. Use - to read the message from the standard input")
	newMessage     = newFlagSet.String("m", "", "Provide a message to describe the issue")
)

func RunNewBug(repo repository.Repo, args []string) error {
	newFlagSet.Parse(args)
	args = newFlagSet.Args()

	var err error

	if len(args) == 0 {
		return errors.New("No title provided")
	}
	if len(args) > 1 {
		return errors.New("Only accepting one title is supported")
	}

	title := args[0]

	if *newMessageFile != "" && *newMessage == "" {
		*newMessage, err = input.FromFile(*newMessageFile)
		if err != nil {
			return err
		}
	}
	if *newMessageFile == "" && *newMessage == "" {
		*newMessage, err = input.LaunchEditor(repo, messageFilename)
		if err != nil {
			return err
		}
	}

	// Note: this is very primitive for now
	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	newbug, err := bug.NewBug()
	if err != nil {
		return err
	}

	createOp := operations.NewCreateOp(author, title, *newMessage)

	newbug.Append(createOp)
	newbug.Commit(repo)

	return nil

}

var newCmd = &Command{
	Usage: func(arg0 string) {
		fmt.Printf("Usage: %s new <title> [<option>...]\n\nOptions:\n", arg0)
		newFlagSet.PrintDefaults()
	},
	RunMethod: RunNewBug,
}
