package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
)

func runBridgeAuthRm(cmd *cobra.Command, args []string) error {
	cred, err := auth.LoadWithPrefix(repo, args[0])
	if err != nil {
		return err
	}

	err = auth.Remove(repo, cred.ID())
	if err != nil {
		return err
	}

	fmt.Printf("credential %s removed\n", cred.ID())
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
