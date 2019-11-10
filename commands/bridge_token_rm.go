package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func runBridgeTokenRm(cmd *cobra.Command, args []string) error {
	token, err := core.LoadTokenPrefix(repo, args[0])
	if err != nil {
		return err
	}

	err = core.RemoveToken(repo, token.ID())
	if err != nil {
		return err
	}

	fmt.Printf("token %s removed\n", token.ID())
	return nil
}

var bridgeTokenRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Short:   "Remove a token.",
	PreRunE: loadRepo,
	RunE:    runBridgeTokenRm,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeTokenCmd.AddCommand(bridgeTokenRmCmd)
}
