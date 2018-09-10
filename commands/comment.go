package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/spf13/cobra"
)

var (
	commentMessageFile string
	commentMessage     string
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

	if commentMessageFile != "" && commentMessage == "" {
		commentMessage, err = input.FromFile(commentMessageFile)
		if err != nil {
			return err
		}
	}

	if commentMessage == "" {
		commentMessage, err = input.BugCommentEditorInput(backend.Repository())
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

	err = b.AddComment(commentMessage)
	if err != nil {
		return err
	}

	return b.Commit()
}

var commentCmd = &cobra.Command{
	Use:   "comment <id>",
	Short: "Add a new comment to a bug",
	RunE:  runComment,
}

func init() {
	RootCmd.AddCommand(commentCmd)

	commentCmd.Flags().SortFlags = false

	commentCmd.Flags().StringVarP(&commentMessageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input",
	)

	commentCmd.Flags().StringVarP(&commentMessage, "message", "m", "",
		"Provide the new message from the command line",
	)
}
