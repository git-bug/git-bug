package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	text "github.com/MichaelMure/go-term-text"
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
	idFmt := text.LeftPadMaxLine(token.ID.Human(), 7, 0)
	valueFmt := text.LeftPadMaxLine(token.Value, 15, 0)
	targetFmt := text.LeftPadMaxLine(token.Target, 7, 0)
	createTimeFmt := text.LeftPadMaxLine(token.CreateTime.Format(time.RFC822), 20, 0)

	fmt.Printf("%s %s %s %s\n",
		idFmt,
		colors.Magenta(targetFmt),
		valueFmt,
		createTimeFmt,
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
