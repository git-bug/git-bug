package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func runLabel(cmd *cobra.Command, args []string) error {
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

	for _, l := range snap.Labels {
		fmt.Println(l)
	}

	return nil
}

var labelCmd = &cobra.Command{
	Use:   "label [<id>]",
	Short: "Display, add or remove labels",
	RunE:  runLabel,
}

func init() {
	RootCmd.AddCommand(labelCmd)

	labelCmd.Flags().SortFlags = false
}
