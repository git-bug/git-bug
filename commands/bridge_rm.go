package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func runBridgeRm(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	err = bridge.RemoveBridge(backend, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Successfully removed bridge configuration %v\n", args[0])
	return nil
}

var bridgeRmCmd = &cobra.Command{
	Use:     "rm <name>",
	Short:   "Delete a configured bridge.",
	PreRunE: loadRepo,
	RunE:    runBridgeRm,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeCmd.AddCommand(bridgeRmCmd)
}
