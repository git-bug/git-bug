package commands

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/input"
	"github.com/spf13/cobra"
)

var (
	newTitle       string
	newMessage     string
	newMessageFile string
)

func runNewBug(cmd *cobra.Command, args []string) error {
	var err error

	if newMessageFile != "" && newMessage == "" {
		newMessage, err = input.FromFile(newMessageFile)
		if err != nil {
			return err
		}
	}

	if newMessage == "" || newTitle == "" {
		newTitle, newMessage, err = input.BugCreateEditorInput(repo, messageFilename, newTitle, newMessage)

		if err == input.ErrEmptyTitle {
			fmt.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	newBug, err := operations.Create(author, newTitle, newMessage)
	if err != nil {
		return err
	}

	err = newBug.Commit(repo)

	if err != nil {
		return err
	}

	fmt.Printf("%s created\n", newBug.HumanId())

	return nil
}

var newCmd = &cobra.Command{
	Use:   "new [<option>...]",
	Short: "Create a new bug",
	RunE:  runNewBug,
}

func init() {
	RootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVarP(&newTitle, "title", "t", "",
		"Provide a title to describe the issue",
	)
	newCmd.Flags().StringVarP(&newMessage, "message", "m", "",
		"Provide a message to describe the issue",
	)
	newCmd.Flags().StringVarP(&newMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)
}
