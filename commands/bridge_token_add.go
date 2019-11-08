package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
)

var (
	bridgeTokenValue  string
	bridgeTokenTarget string
)

func runBridgeTokenAdd(cmd *cobra.Command, args []string) error {
	token := core.NewToken(bridgeTokenValue, bridgeTokenTarget)
	if err := token.Validate(); err != nil {
		return errors.Wrap(err, "invalid token")
	}

	return core.StoreToken(repo, token)
}

var bridgeTokenAddCmd = &cobra.Command{
	Use:     "add",
	Short:   "Create and store a new token",
	PreRunE: loadRepo,
	RunE:    runBridgeTokenAdd,
	Args:    cobra.NoArgs,
}

func init() {
	bridgeTokenCmd.AddCommand(bridgeTokenAddCmd)
	bridgeTokenAddCmd.Flags().StringVarP(&bridgeTokenValue, "value", "v", "", "")
	bridgeTokenAddCmd.Flags().StringVarP(&bridgeTokenTarget, "target", "t", "", "")
	bridgeTokenAddCmd.Flags().SortFlags = false
}
