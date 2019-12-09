package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
)

func runBridgeAuthShow(cmd *cobra.Command, args []string) error {
	cred, err := auth.LoadWithPrefix(repo, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Id: %s\n", cred.ID())
	fmt.Printf("Target: %s\n", cred.Target())
	fmt.Printf("Kind: %s\n", cred.Kind())
	fmt.Printf("Creation: %s\n", cred.CreateTime().Format(time.RFC822))

	switch cred := cred.(type) {
	case *auth.Token:
		fmt.Printf("Value: %s\n", cred.Value)
	}

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
