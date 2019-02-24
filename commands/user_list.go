package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runUserLs(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	for _, i := range backend.AllIdentityExcerpt() {
		fmt.Printf("%s %s\n",
			colors.Cyan(i.HumanId()),
			i.DisplayName(),
		)
	}

	return nil
}

var userLsCmd = &cobra.Command{
	Use:     "ls",
	Short:   "List identities.",
	PreRunE: loadRepo,
	RunE:    runUserLs,
}

func init() {
	userCmd.AddCommand(userLsCmd)
	userLsCmd.Flags().SortFlags = false
}
