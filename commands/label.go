package commands

import (
	"errors"
	"os"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/spf13/cobra"
)

var labelRemove bool

func runLabel(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	if len(args) == 1 {
		return errors.New("You must provide a label")
	}

	prefix := args[0]

	var add, remove []string

	if labelRemove {
		remove = args[1:]
	} else {
		add = args[1:]
	}

	b, err := bug.FindLocalBug(repo, prefix)
	if err != nil {
		return err
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	err = operations.ChangeLabels(os.Stdout, b, author, add, remove)

	if err != nil {
		return err
	}

	return b.Commit(repo)
}

var labelCmd = &cobra.Command{
	Use:   "label [<option>...] <id> [<label>...]",
	Short: "Manipulate bug's label",
	RunE:  runLabel,
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().BoolVarP(&labelRemove, "remove", "r", false,
		"Remove a label",
	)
}
