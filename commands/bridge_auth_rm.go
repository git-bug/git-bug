package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func runBridgeAuthRm(cmd *cobra.Command, args []string) error {
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

var bridgeAuthRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Short:   "Remove a credential.",
	PreRunE: loadRepo,
	RunE:    runBridgeAuthRm,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeAuthCmd.AddCommand(bridgeAuthRmCmd)
}
