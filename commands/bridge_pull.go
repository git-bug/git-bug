package commands

import (
	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runBridgePull(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	var b *core.Bridge

	if len(args) == 0 {
		b, err = bridge.DefaultBridge(backend)
	} else {
		b, err = bridge.NewBridgeFromFullName(backend, args[0])
	}

	if err != nil {
		return err
	}

	err = b.ImportAll()
	if err != nil {
		return err
	}

	return nil
}

var bridgePullCmd = &cobra.Command{
	Use:     "pull [<name>]",
	Short:   "Pull updates",
	PreRunE: loadRepo,
	RunE:    runBridgePull,
}

func init() {
	bridgeCmd.AddCommand(bridgePullCmd)
}
