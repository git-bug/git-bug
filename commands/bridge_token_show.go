package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

func runBridgeTokenShow(cmd *cobra.Command, args []string) error {
	token, err := core.LoadTokenPrefix(repo, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Id: %s\n", token.ID())
	fmt.Printf("Value: %s\n", token.Value)
	fmt.Printf("Target: %s\n", token.Target)
	fmt.Printf("Creation: %s\n", token.CreateTime.Format(time.RFC822))

	return nil
}

var bridgeTokenShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Display a token.",
	PreRunE: loadRepo,
	RunE:    runBridgeTokenShow,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeTokenCmd.AddCommand(bridgeTokenShowCmd)
}
