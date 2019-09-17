package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

var (
	bridgeTokenAll        bool
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

	var tokens []string
	if bridgeTokenLocalOnly || bridgeTokenAll {
		localTokens, err := core.ListTokens(backend)
		if err != nil {
			return err
		}
		tokens = localTokens
	}

	if bridgeTokenGlobalOnly || bridgeTokenAll {
		globalTokens, err := core.ListGlobalTokens(backend)
		if err != nil {
			return err
		}
		tokens = append(tokens, globalTokens...)
	}

	for _, c := range tokens {
		fmt.Println(c)
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
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenAll, "all", "a", false, "")
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenLocalOnly, "local", "l", true, "")
	bridgeTokenCmd.Flags().BoolVarP(&bridgeTokenGlobalOnly, "global", "g", false, "")
	bridgeTokenCmd.Flags().SortFlags = false
}
