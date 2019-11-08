package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func runBridgeTokenRm(cmd *cobra.Command, args []string) error {
	return core.RemoveToken(repo, args[0])
}

var bridgeTokenRmCmd = &cobra.Command{
	Use:     "rm",
	Short:   "Remove token by Id.",
	PreRunE: loadRepo,
	RunE:    runBridgeTokenRm,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeTokenCmd.AddCommand(bridgeTokenRmCmd)
}
