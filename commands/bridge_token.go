package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	text "github.com/MichaelMure/go-term-text"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/util/colors"
)

func runTokenBridge(cmd *cobra.Command, args []string) error {
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
	valueFmt := text.LeftPadMaxLine(token.Value, 15, 0)
	targetFmt := text.LeftPadMaxLine(token.Target, 7, 0)
	createTimeFmt := text.LeftPadMaxLine(token.CreateTime.Format(time.RFC822), 20, 0)

	fmt.Printf("%s %s %s %s\n",
		token.ID().Human(),
		colors.Magenta(targetFmt),
		valueFmt,
		createTimeFmt,
	)
}

var bridgeTokenCmd = &cobra.Command{
	Use:     "token",
	Short:   "List all known tokens.",
	PreRunE: loadRepo,
	RunE:    runTokenBridge,
	Args:    cobra.NoArgs,
}

func init() {
	bridgeCmd.AddCommand(bridgeTokenCmd)
	bridgeTokenCmd.Flags().SortFlags = false
}
