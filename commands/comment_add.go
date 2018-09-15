package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	commentAddMessageFile string
	commentAddMessage     string
)

func runCommentAdd(cmd *cobra.Command, args []string) error {
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

	if commentAddMessageFile != "" && commentAddMessage == "" {
		commentAddMessage, err = input.FromFile(commentAddMessageFile)
		if err != nil {
			return err
		}
	}

	if commentAddMessage == "" {
		commentAddMessage, err = input.BugCommentEditorInput(backend.Repository())
		if err == input.ErrEmptyMessage {
			fmt.Println("Empty message, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	err = b.AddComment(commentAddMessage)
	if err != nil {
		return err
	}

	return b.Commit()
}

var commentAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a new comment to a bug",
	RunE:  runCommentAdd,
}

func init() {
	commentCmd.AddCommand(commentAddCmd)

	commentCmd.Flags().SortFlags = false

	commentCmd.Flags().StringVarP(&commentAddMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)

	commentCmd.Flags().StringVarP(&commentAddMessage, "message", "m", "",
		"Provide the new message from the command line",
	)
}
