package commands

import (
	"errors"
	"fmt"
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

	b, err := bug.FindBug(repo, prefix)
	if err != nil {
		return err
	}

	author, err := bug.GetUser(repo)
	if err != nil {
		return err
	}

	var added, removed []bug.Label

	snap := b.Compile()

	for _, arg := range args[1:] {
		label := bug.Label(arg)

		if labelRemove {
			// check for duplicate
			if labelExist(removed, label) {
				fmt.Printf("label \"%s\" is a duplicate\n", arg)
				continue
			}

			// check that the label actually exist
			if !labelExist(snap.Labels, label) {
				fmt.Printf("label \"%s\" doesn't exist on this bug\n", arg)
				continue
			}

			removed = append(removed, label)
		} else {
			// check for duplicate
			if labelExist(added, label) {
				fmt.Printf("label \"%s\" is a duplicate\n", arg)
				continue
			}

			// check that the label doesn't already exist
			if labelExist(snap.Labels, label) {
				fmt.Printf("label \"%s\" is already set on this bug\n", arg)
				continue
			}

			added = append(added, label)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return errors.New("no label added or removed")
	}

	labelOp := operations.NewLabelChangeOperation(author, added, removed)

	b.Append(labelOp)

	err = b.Commit(repo)

	return err
}

func labelExist(labels []bug.Label, label bug.Label) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}

	return false
}

var labelCmd = &cobra.Command{
	Use:   "label [<option>...] <id> [<label>...]",
	Short: "Manipulate bug's label",
	RunE:  runLabel,
}

func init() {
	rootCmd.AddCommand(labelCmd)

	labelCmd.Flags().BoolVarP(&labelRemove, "remove", "r", false,
		"Remove a label",
	)
}
