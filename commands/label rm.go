package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runLabelRm(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]
	remove := args[1:]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	changes, err := b.ChangeLabels(nil, remove)

	for _, change := range changes {
		fmt.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}

var labelRmCmd = &cobra.Command{
	Use:   "rm <id> [<label>...]",
	Short: "Remove a label from a bug",
	RunE:  runLabelRm,
}

func init() {
	labelCmd.AddCommand(labelRmCmd)
}
