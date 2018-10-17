package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/spf13/cobra"
)

func runBridge(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	configured, err := bridge.ConfiguredBridges(backend)
	if err != nil {
		return err
	}

	for _, c := range configured {
		fmt.Println(c)
	}

	return nil
}

var bridgeCmd = &cobra.Command{
	Use:     "bridge",
	Short:   "Configure and use bridges to other bug trackers",
	PreRunE: loadRepo,
	RunE:    runBridge,
	Args:    cobra.NoArgs,
}

func init() {
	RootCmd.AddCommand(bridgeCmd)
}
