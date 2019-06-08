package commands

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func runBridgePush(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var b *core.Bridge

	if len(args) == 0 {
		b, err = bridge.DefaultBridge(backend)
	} else {
		b, err = bridge.NewBridgeFromFullName(backend, args[0])
	}

	if err != nil {
		return err
	}

	// TODO: by default export only new events
	err = b.ExportAll(time.Time{})
	if err != nil {
		return err
	}

	return nil
}

var bridgePushCmd = &cobra.Command{
	Use:     "push [<name>]",
	Short:   "Push updates.",
	PreRunE: loadRepo,
	RunE:    runBridgePush,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	bridgeCmd.AddCommand(bridgePushCmd)
}
