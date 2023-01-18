package bugcmd

import (
	"github.com/spf13/cobra"

	buginput "github.com/MichaelMure/git-bug/commands/bug/input"
	"github.com/MichaelMure/git-bug/commands/execenv"
)

type bugCommentEditOptions struct {
	messageFile    string
	message        string
	nonInteractive bool
}

func newBugCommentEditCommand(env *execenv.Env) *cobra.Command {
	options := bugCommentEditOptions{}

	cmd := &cobra.Command{
		Use:     "edit [COMMENT_ID]",
		Short:   "Edit an existing comment on a bug",
		Args:    cobra.ExactArgs(1),
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugCommentEdit(env, options, args)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")

	flags.StringVarP(&options.message, "message", "m", "",
		"Provide the new message from the command line")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugCommentEdit(env *execenv.Env, opts bugCommentEditOptions, args []string) error {
	b, commentId, err := env.Backend.Bugs().ResolveComment(args[0])
	if err != nil {
		return err
	}

	if opts.messageFile != "" && opts.message == "" {
		opts.message, err = buginput.BugCommentFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if opts.messageFile == "" && opts.message == "" {
		if opts.nonInteractive {
			env.Err.Println("No message given. Use -m or -F option to specify a message. Aborting.")
			return nil
		}
		opts.message, err = buginput.BugCommentEditorInput(env.Backend, "")
		if err == buginput.ErrEmptyMessage {
			env.Err.Println("Empty message, aborting.")
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
