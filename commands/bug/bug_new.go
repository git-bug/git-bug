package bugcmd

import (
	"github.com/spf13/cobra"

	buginput "github.com/git-bug/git-bug/commands/bug/input"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/util/text"
)

type bugNewOptions struct {
	title          string
	message        string
	messageFile    string
	nonInteractive bool
}

func newBugNewCommand(env *execenv.Env) *cobra.Command {
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
		"Take the title and message from the given file (the first line is the title, unless --title is given). Use - to read from the standard input")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	cmd.MarkFlagsMutuallyExclusive("message", "file")

	return cmd
}

func runBugNew(env *execenv.Env, opts bugNewOptions) error {
	var err error
	if opts.messageFile != "" {
		opts.title, opts.message, err = buginput.BugCreateFileInput(opts.messageFile, opts.title)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.messageFile == "" && (opts.message == "" || opts.title == "") {
		opts.title, opts.message, err = buginput.BugCreateEditorInput(env.Backend, opts.title, opts.message)

		if err == buginput.ErrEmptyTitle {
			env.Out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	b, _, err := env.Backend.Bugs().New(
		text.CleanupOneLine(opts.title),
		text.Cleanup(opts.message),
	)
	if err != nil {
		return err
	}

	env.Out.Printf("%s created\n", b.Id().Human())

	return nil
}
