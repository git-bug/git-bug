package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/text"
)

type commentAddOptions struct {
	messageFile    string
	message        string
	nonInteractive bool
}

func newCommentAddCommand() *cobra.Command {
	env := newEnv()
	options := commentAddOptions{}

	cmd := &cobra.Command{
		Use:      "add [ID]",
		Short:    "Add a new comment to a bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommentAdd(env, options, args)
		},
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

func runCommentAdd(env *Env, opts commentAddOptions, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
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
		if opts.nonInteractive {
			env.err.Println("No message given. Use -m or -F option to specify a message. Aborting.")
			return nil
		}
		opts.message, err = input.BugCommentEditorInput(env.backend, "")
		if err == input.ErrEmptyMessage {
			env.err.Println("Empty message, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	_, err = b.AddComment(text.Cleanup(opts.message))
	if err != nil {
		return err
	}

	return b.Commit()
}
