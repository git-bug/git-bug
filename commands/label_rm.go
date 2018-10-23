package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/cleaner"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func runLabelRm(cmd *cobra.Command, args []string) error {
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

	changes, err := b.ChangeLabels(nil, args)

	for _, change := range changes {
		fmt.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}

var labelRmCmd = &cobra.Command{
	Use:     "rm [<id>] <label>[...]",
	Short:   "Remove a label",
	PreRunE: loadRepo,
	RunE:    runLabelRm,
}

func init() {
	labelCmd.AddCommand(labelRmCmd)
}
