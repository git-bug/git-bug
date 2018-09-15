package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func runTitle(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) > 1 {
		return errors.New("Only one bug id is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	fmt.Println(snap.Title)

	return nil
}

var titleCmd = &cobra.Command{
	Use:   "title <id>",
	Short: "Display a bug's title",
	RunE:  runTitle,
}

func init() {
	RootCmd.AddCommand(titleCmd)

	titleCmd.Flags().SortFlags = false
}
