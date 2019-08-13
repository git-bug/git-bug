package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/spf13/cobra"
)

func runComment(cmd *cobra.Command, args []string) error {
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

	commentsTextOutput(snap.Comments)

	return nil
}

func commentsTextOutput(comments []bug.Comment) {
	for i, comment := range comments {
		if i != 0 {
			fmt.Println()
		}

		fmt.Printf("Author: %s\n", colors.Magenta(comment.Author.DisplayName()))
		fmt.Printf("Id: %s\n", colors.Cyan(comment.Id().Human()))
		fmt.Printf("Date: %s\n\n", comment.FormatTime())
		fmt.Println(text.LeftPad(comment.Message, 4))
	}
}

var commentCmd = &cobra.Command{
	Use:     "comment [<id>]",
	Short:   "Display or add comments to a bug.",
	PreRunE: loadRepo,
	RunE:    runComment,
}

func init() {
	RootCmd.AddCommand(commentCmd)

	commentCmd.Flags().SortFlags = false
}
