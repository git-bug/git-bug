package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func runTitle(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	fmt.Println(snap.Title)

	return nil
}

var titleCmd = &cobra.Command{
	Use:     "title [<id>]",
	Short:   "Display or change a title",
	PreRunE: loadRepo,
	RunE:    runTitle,
}

func init() {
	RootCmd.AddCommand(titleCmd)

	titleCmd.Flags().SortFlags = false
}
