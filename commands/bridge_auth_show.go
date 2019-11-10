package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func runBridgeAuthShow(cmd *cobra.Command, args []string) error {
	token, err := core.LoadTokenPrefix(repo, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Id: %s\n", token.ID())
	fmt.Printf("Target: %s\n", token.Target)
	fmt.Printf("Type: token\n")
	fmt.Printf("Value: %s\n", token.Value)
	fmt.Printf("Creation: %s\n", token.CreateTime.Format(time.RFC822))

	return nil
}

var bridgeAuthShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Display an authentication credential.",
	PreRunE: loadRepo,
	RunE:    runBridgeAuthShow,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeAuthCmd.AddCommand(bridgeAuthShowCmd)
}
