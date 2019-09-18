package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func runBridgeTokenRm(cmd *cobra.Command, args []string) error {
	err := core.RemoveToken(repo, args[0])
	if err == nil {
		return nil
	}

	err = core.RemoveGlobalToken(repo, args[0])
	return err
}

var bridgeTokenRmCmd = &cobra.Command{
	Use:     "rm",
	Short:   "Configure and use bridge tokens.",
	PreRunE: loadRepo,
	RunE:    runBridgeTokenRm,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeTokenCmd.AddCommand(bridgeTokenRmCmd)
}
