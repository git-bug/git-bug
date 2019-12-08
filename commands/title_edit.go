package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

var (
	titleEditTitle string
)

func runTitleEdit(cmd *cobra.Command, args []string) error {
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

	snap := b.Snapshot()

	if titleEditTitle == "" {
		titleEditTitle, err = input.BugTitleEditorInput(repo, snap.Title)
		if err == input.ErrEmptyTitle {
			fmt.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	if titleEditTitle == snap.Title {
		fmt.Println("No change, aborting.")
	}

	_, err = b.SetTitle(titleEditTitle)
	if err != nil {
		return err
	}

	return b.Commit()
}

var titleEditCmd = &cobra.Command{
	Use:     "edit [<id>]",
	Short:   "Edit a title of a bug.",
	PreRunE: loadRepoEnsureUser,
	RunE:    runTitleEdit,
}

func init() {
	titleCmd.AddCommand(titleEditCmd)

	titleEditCmd.Flags().SortFlags = false

	titleEditCmd.Flags().StringVarP(&titleEditTitle, "title", "t", "",
		"Provide a title to describe the issue",
	)
}
