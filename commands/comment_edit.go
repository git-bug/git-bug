package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/input"
)

type commentEditOptions struct {
	messageFile string
	message     string
}

func newCommentEditCommand() *cobra.Command {
	env := newEnv()
	options := commentEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit <commentid>",
		Short:    "Edit an existing comment on a bug.",
		Args:     cobra.ExactArgs(1),
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommentEdit(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")

	flags.StringVarP(&options.message, "message", "m", "",
		"Provide the new message from the command line")

	return cmd
}

func runCommentEdit(env *Env, opts commentEditOptions, args []string) error {
	b, commentId, err := env.backend.ResolveComment(args[0])
	if err != nil {
		return err
	}

	if opts.messageFile != "" && opts.message == "" {
		opts.message, err = input.BugCommentFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if opts.messageFile == "" && opts.message == "" {
		opts.message, err = input.BugCommentEditorInput(env.backend, "")
		if err == input.ErrEmptyMessage {
			env.err.Println("Empty message, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	_, err = b.EditComment(commentId, opts.message)
	if err != nil {
		return err
	}

	return b.Commit()
}
