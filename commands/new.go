package commands

import (
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/commands/input"
	"github.com/spf13/cobra"
)

var (
	newMessageFile string
	newMessage     string
)

func runNewBug(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) == 0 {
		return errors.New("No title provided")
	}
	if len(args) > 1 {
		return errors.New("Only accepting one title is supported")
	}

	title := args[0]

	if newMessageFile != "" && newMessage == "" {
		newMessage, err = input.FromFile(newMessageFile)
		if err != nil {
			return err
		}
	}
	if newMessageFile == "" && newMessage == "" {
		newMessage, err = input.LaunchEditor(repo, messageFilename)
		if err != nil {
			return err
		}
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	newBug, err := bug.NewBug()
	if err != nil {
		return err
	}

	createOp := operations.NewCreateOp(author, title, newMessage)

	newBug.Append(createOp)

	err = newBug.Commit(repo)

	fmt.Println(newBug.HumanId())

	return err

}

var newCmd = &cobra.Command{
	Use:   "new <title> [<option>...]",
	Short: "Create a new bug",
	RunE:  runNewBug,
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVarP(&newMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)
	newCmd.Flags().StringVarP(&newMessage, "message", "m", "",
		"Provide a message to describe the issue",
	)
}
