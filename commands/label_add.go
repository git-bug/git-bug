package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runLabelAdd(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	changes, err := b.ChangeLabels(args, nil)

	for _, change := range changes {
		fmt.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}

var labelAddCmd = &cobra.Command{
	Use:     "add [<id>] <label>[...]",
	Short:   "Add a label",
	PreRunE: loadRepo,
	RunE:    runLabelAdd,
}

func init() {
	labelCmd.AddCommand(labelAddCmd)
}
