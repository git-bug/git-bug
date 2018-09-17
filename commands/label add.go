package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runLabelAdd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]
	add := args[1:]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	changes, err := b.ChangeLabels(add, nil)
	if err != nil {
		return err
	}

	for _, change := range changes {
		fmt.Println(change)
	}

	return b.Commit()
}

var labelAddCmd = &cobra.Command{
	Use:   "add <id> [<label>...]",
	Short: "Add a label to a bug",
	RunE:  runLabelAdd,
}

func init() {
	// labelCmd.AddCommand(labelAddCmd)
}
