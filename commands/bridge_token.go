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
	bridgeTokenLocalOnly  bool
	bridgeTokenGlobalOnly bool
)

func runTokenBridge(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var tokens []*core.Token
	if !bridgeTokenGlobalOnly {
		localTokens, err := core.ListTokens(backend)
		if err != nil {
			return err
		}

		for _, id := range localTokens {
			token, err := core.GetToken(repo, id)
			if err != nil {
				return err
			}
			tokens = append(tokens, token)
		}
	}

	if !bridgeTokenLocalOnly {
		globalTokens, err := core.ListGlobalTokens(backend)
		if err != nil {
			return err
		}

		for _, id := range globalTokens {
			token, err := core.GetGlobalToken(repo, id)
			if err != nil {
				return err
			}
			tokens = append(tokens, token)
		}
	}

	for _, token := range tokens {
		valueFmt := text.LeftPadMaxLine(token.Value, 20, 0)
		targetFmt := text.LeftPadMaxLine(token.Target, 8, 0)
		scopesFmt := text.LeftPadMaxLine(strings.Join(token.Scopes, ","), 20, 0)

		fmt.Printf("%s %s %s %s\n",
			valueFmt,
			colors.Magenta(targetFmt),
			colors.Yellow(token.Global),
			scopesFmt,
		)
	}
	return nil
}

var bridgeTokenCmd = &cobra.Command{
	Use:     "token",
	Short:   "Configure and use bridge tokens.",
	PreRunE: loadRepo,
	RunE:    runTokenBridge,
	Args:    cobra.NoArgs,
}

func init() {
	bridgeCmd.AddCommand(bridgeTokenCmd)
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenLocalOnly, "local", "l", false, "")
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenGlobalOnly, "global", "g", false, "")
	bridgeTokenCmd.Flags().SortFlags = false
}
