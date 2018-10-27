package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runStatus(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	fmt.Println(snap.Status)

	return nil
}

var statusCmd = &cobra.Command{
	Use:     "status [<id>]",
	Short:   "Display or change a bug status",
	PreRunE: loadRepo,
	RunE:    runStatus,
}

func init() {
	RootCmd.AddCommand(statusCmd)
}
