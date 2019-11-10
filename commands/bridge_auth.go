package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	text "github.com/MichaelMure/go-term-text"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/util/colors"
)

func runBridgeAuth(cmd *cobra.Command, args []string) error {
	tokens, err := core.ListTokens(repo)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		token, err := core.LoadToken(repo, token)
		if err != nil {
			return err
		}
		printToken(token)
	}

	return nil
}

func printToken(token *core.Token) {
	targetFmt := text.LeftPadMaxLine(token.Target, 10, 0)

	fmt.Printf("%s %s %s %s\n",
		colors.Cyan(token.ID().Human()),
		colors.Yellow(targetFmt),
		colors.Magenta("token"),
		token.Value,
	)
}

var bridgeAuthCmd = &cobra.Command{
	Use:     "auth",
	Short:   "List all known bridge authentication credentials.",
	PreRunE: loadRepo,
	RunE:    runBridgeAuth,
	Args:    cobra.NoArgs,
}

func init() {
	bridgeCmd.AddCommand(bridgeAuthCmd)
	bridgeAuthCmd.Flags().SortFlags = false
}
