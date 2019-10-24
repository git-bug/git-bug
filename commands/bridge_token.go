package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/MichaelMure/git-bug/util/text"
)

var (
	bridgeTokenLocal  bool
	bridgeTokenGlobal bool
)

func runTokenBridge(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	tokens, err := core.ListTokens(backend)
	if err != nil {
		return err
	}

	for token, global := range tokens {
		// TODO: filter tokens using flags
		getTokenFn := core.GetToken
		if global {
			getTokenFn = core.GetGlobalToken
		}

		token, err := getTokenFn(repo, token)
		if err != nil {
			return err
		}
		printToken(token)
	}

	return nil
}

func printToken(token *core.Token) {
	idFmt := text.LeftPadMaxLine(token.HumanId(), 7, 0)
	valueFmt := text.LeftPadMaxLine(token.Value, 8, 0)
	targetFmt := text.LeftPadMaxLine(token.Target, 8, 0)
	scopesFmt := text.LeftPadMaxLine(strings.Join(token.Scopes, ","), 20, 0)

	fmt.Printf("%s %s %s %s %s\n",
		idFmt,
		valueFmt,
		colors.Magenta(targetFmt),
		colors.Yellow(token.Kind()),
		scopesFmt,
	)
}

var bridgeTokenCmd = &cobra.Command{
	Use:     "token",
	Short:   "List all stored tokens.",
	PreRunE: loadRepo,
	RunE:    runTokenBridge,
	Args:    cobra.NoArgs,
}

func init() {
	bridgeCmd.AddCommand(bridgeTokenCmd)
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenLocal, "local", "l", false, "")
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenGlobal, "global", "g", false, "")
	bridgeTokenCmd.Flags().SortFlags = false
}
