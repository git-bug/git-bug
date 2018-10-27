package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runSelect(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	prefix := args[0]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	err = _select.Select(backend, b.Id())
	if err != nil {
		return err
	}

	fmt.Printf("selected bug %s: %s\n", b.HumanId(), b.Snapshot().Title)

	return nil
}

var selectCmd = &cobra.Command{
	Use:   "select <id>",
	Short: "Select a bug for implicit use in future commands",
	Example: `git bug select 2f15
git bug comment
git bug status
`,
	PreRunE: loadRepo,
	RunE:    runSelect,
}

func init() {
	RootCmd.AddCommand(selectCmd)
	selectCmd.Flags().SortFlags = false
}
