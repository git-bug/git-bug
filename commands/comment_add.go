package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/input"
	"github.com/spf13/cobra"
)

var (
	commentAddMessageFile string
	commentAddMessage     string
)

func runCommentAdd(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	if commentAddMessageFile != "" && commentAddMessage == "" {
		commentAddMessage, err = input.FromFile(commentAddMessageFile)
		if err != nil {
			return err
		}
	}

	if commentAddMessage == "" {
		commentAddMessage, err = input.BugCommentEditorInput(backend, "")
		if err == input.ErrEmptyMessage {
			fmt.Println("Empty message, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	b, args, err := _select.ResolveBug(backend, args)
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
	Use:     "add [<id>]",
	Short:   "Add a new comment",
	PreRunE: loadRepo,
	RunE:    runCommentAdd,
}

func init() {
	commentCmd.AddCommand(commentAddCmd)

	commentAddCmd.Flags().SortFlags = false

	commentAddCmd.Flags().StringVarP(&commentAddMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)

	commentAddCmd.Flags().StringVarP(&commentAddMessage, "message", "m", "",
		"Provide the new message from the command line",
	)
}
