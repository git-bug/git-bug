package bugcmd

import (
	"github.com/spf13/cobra"

	buginput "github.com/MichaelMure/git-bug/commands/bug/input"
	"github.com/MichaelMure/git-bug/commands/bug/select"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/util/text"
)

type bugCommentNewOptions struct {
	messageFile    string
	message        string
	nonInteractive bool
}

func newBugCommentNewCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := bugCommentNewOptions{}

	cmd := &cobra.Command{
		Use:     "new [BUG_ID]",
		Short:   "Add a new comment to a bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugCommentNew(env, options, args)
		}),
		ValidArgsFunction: completion.Bug(env),
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

func runBugCommentNew(env *execenv.Env, opts bugCommentNewOptions, args []string) error {
	b, args, err := _select.ResolveBug(env.Backend, args)
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

	_, _, err = b.AddComment(text.Cleanup(opts.message))
	if err != nil {
		return err
	}

	return b.Commit()
}
