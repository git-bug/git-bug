package commands

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/cleaner"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func runStatusOpen(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	cleaner.Register(backend.Close)

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	err = b.Open()
	if err != nil {
		return err
	}

	return b.Commit()
}

var openCmd = &cobra.Command{
	Use:     "open [<id>]",
	Short:   "Mark a bug as open",
	PreRunE: loadRepo,
	RunE:    runStatusOpen,
}

func init() {
	statusCmd.AddCommand(openCmd)
}
