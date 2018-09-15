package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/spf13/cobra"
)

var (
	titleEditTitle string
)

func runTitleEdit(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) > 1 {
		return errors.New("Only one bug id is supported")
	}

	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	prefix := args[0]

	b, err := backend.ResolveBugPrefix(prefix)
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

	err = b.SetTitle(titleEditTitle)
	if err != nil {
		return err
	}

	return b.Commit()
}

var titleEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a bug title",
	RunE:  runTitleEdit,
}

func init() {
	titleCmd.AddCommand(titleEditCmd)

	titleEditCmd.Flags().SortFlags = false

	titleEditCmd.Flags().StringVarP(&titleEditTitle, "title", "t", "",
		"Provide a title to describe the issue",
	)
}
