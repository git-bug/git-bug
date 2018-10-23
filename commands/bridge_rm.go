package commands

import (
	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/cleaner"
	"github.com/spf13/cobra"
)

func runBridgeRm(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	cleaner.Register(backend.Close)

	err = bridge.RemoveBridges(backend, args[0])
	if err != nil {
		return err
	}

	return nil
}

var bridgeRmCmd = &cobra.Command{
	Use:     "rm name <name>",
	Short:   "Delete a configured bridge",
	PreRunE: loadRepo,
	RunE:    runBridgeRm,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeCmd.AddCommand(bridgeRmCmd)
}
