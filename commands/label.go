package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/operations"
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

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]

	var add, remove []string

	if labelRemove {
		remove = args[1:]
	} else {
		add = args[1:]
	}

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	changes, err := b.ChangeLabels(add, remove)

	for _, change := range changes {
		switch change.Status {
		case operations.LabelChangeAdded:
			fmt.Printf("label %s added\n", change.Label)
		case operations.LabelChangeRemoved:
			fmt.Printf("label %s removed\n", change.Label)
		case operations.LabelChangeDuplicateInOp:
			fmt.Printf("label %s is a duplicate\n", change.Label)
		case operations.LabelChangeAlreadySet:
			fmt.Printf("label %s was already set\n", change.Label)
		case operations.LabelChangeDoesntExist:
			fmt.Printf("label %s doesn't exist on this bug\n", change.Label)
		default:
			panic(fmt.Sprintf("unknown label change status %v", change.Status))
		}
	}

	if err != nil {
		return err
	}

	return b.Commit()
}

var labelCmd = &cobra.Command{
	Use:   "label <id> [<label>...]",
	Short: "Manipulate bug's label",
	RunE:  runLabel,
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().SortFlags = false

	labelCmd.Flags().BoolVarP(&labelRemove, "remove", "r", false,
		"Remove a label",
	)
}
