package commands

import (
	"errors"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/commands/input"
	"github.com/spf13/cobra"
)

var (
	commentMessageFile string
	commentMessage     string
)

func runComment(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) > 1 {
		return errors.New("Only one bug id is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	prefix := args[0]

	if commentMessageFile != "" && commentMessage == "" {
		commentMessage, err = input.FromFile(commentMessageFile)
		if err != nil {
			return err
		}
	}
	if commentMessageFile == "" && commentMessage == "" {
		commentMessage, err = input.LaunchEditor(repo, messageFilename)
		if err != nil {
			return err
		}
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	b, err := bug.FindBug(repo, prefix)
	if err != nil {
		return err
	}

	addCommentOp := operations.NewAddCommentOp(author, commentMessage)

	b.Append(addCommentOp)

	err = b.Commit(repo)

	return err
}

var commentCmd = &cobra.Command{
	Use:   "comment <id> [<options>...]",
	Short: "Add a new comment to a bug",
	RunE:  runComment,
}

func init() {
	RootCmd.AddCommand(commentCmd)

	commentCmd.Flags().StringVarP(&commentMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)

	commentCmd.Flags().StringVarP(&commentMessage, "message", "m", "",
		"Provide the new message from the command line",
	)
}
