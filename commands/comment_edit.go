package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runCommentEdit(cmd *cobra.Command, args []string) error {

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	for _, id := range backend.AllBugsIds() {
		b, err := backend.ResolveBugPrefix(id)
		if err != nil {
			return err
		}
		snap := b.Snapshot()

		for _, comment := range snap.Comments {

			if args[0] == comment.HumanId() {

				commentEdit, err := input.BugCommentEditorInput(repo, comment.Message)
				if err == input.ErrEmptyMessage {
					fmt.Println("Empty message, aborting.")
					return nil
				}
				if commentEdit == comment.Message {
					fmt.Println("No changes found, aborting.")
					return nil
				}
				_, err = b.EditComment(git.Hash(comment.Id()), commentEdit)
				if err != nil {
					return err
				}
				fmt.Println("Comment edited successfully")
				return b.Commit()
			}
		}
	}
	return fmt.Errorf("no bug found with matching Id")

}

var commentEditCmd = &cobra.Command{
	Use:     "edit [<id>]",
	Short:   "Edit a comment of a bug.",
	PreRunE: loadRepo,
	RunE:    runCommentEdit,
}

func init() {
	commentCmd.AddCommand(commentEditCmd)
	commentEditCmd.Flags().SortFlags = false
}
