package commands

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func runDeselect(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	err = _select.Clear(backend)
	if err != nil {
		return err
	}

	return nil
}

var deselectCmd = &cobra.Command{
	Use:   "deselect",
	Short: "Clear the implicitly selected bug",
	Example: `git bug select 2f15
git bug comment
git bug status
git bug deselect
`,
	RunE: runDeselect,
}

func init() {
	RootCmd.AddCommand(deselectCmd)
	deselectCmd.Flags().SortFlags = false
}
