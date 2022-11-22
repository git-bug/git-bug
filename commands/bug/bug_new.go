package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/commands/input"
	"github.com/MichaelMure/git-bug/util/text"
)

type bugNewOptions struct {
	title          string
	message        string
	messageFile    string
	nonInteractive bool
}

func newBugNewCommand() *cobra.Command {
	env := execenv.NewEnv()
	options := bugNewOptions{}

	cmd := &cobra.Command{
		Use:     "new",
		Short:   "Create a new bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugNew(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue")
	flags.StringVarP(&options.message, "message", "m", "",
		"Provide a message to describe the issue")
	flags.StringVarP(&options.messageFile, "file", "F", "",
		"Take the message from the given file. Use - to read the message from the standard input")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBugNew(env *execenv.Env, opts bugNewOptions) error {
	var err error
	if opts.messageFile != "" && opts.message == "" {
		opts.title, opts.message, err = input.BugCreateFileInput(opts.messageFile)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.messageFile == "" && (opts.message == "" || opts.title == "") {
		opts.title, opts.message, err = input.BugCreateEditorInput(env.Backend, opts.title, opts.message)

		if err == input.ErrEmptyTitle {
			env.Out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	b, _, err := env.Backend.NewBug(
		text.CleanupOneLine(opts.title),
		text.Cleanup(opts.message),
	)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", b.Id().Human())

	return nil
}
