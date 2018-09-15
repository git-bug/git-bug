package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/text"
	"github.com/spf13/cobra"
)

func runComment(cmd *cobra.Command, args []string) error {
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

	commentsTextOutput(snap.Comments)

	return nil
}

func commentsTextOutput(comments []bug.Comment) {
	for i, comment := range comments {
		if i != 0 {
			fmt.Println()
		}

		fmt.Printf("Author: %s\n", colors.Magenta(comment.Author))
		fmt.Printf("Date: %s\n\n", comment.FormatTime())
		fmt.Println(text.LeftPad(comment.Message, 4))
	}
}

var commentCmd = &cobra.Command{
	Use:   "comment <id>",
	Short: "Show a bug's comments",
	RunE:  runComment,
}

func init() {
	RootCmd.AddCommand(commentCmd)

	commentCmd.Flags().SortFlags = false
}
