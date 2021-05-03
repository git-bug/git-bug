package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/text"
)

type titleEditOptions struct {
	title          string
	nonInteractive bool
}

func newTitleEditCommand() *cobra.Command {
	env := newEnv()
	options := titleEditOptions{}

	cmd := &cobra.Command{
		Use:      "edit [ID]",
		Short:    "Edit a title of a bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTitleEdit(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.title, "title", "t", "",
		"Provide a title to describe the issue",
	)
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runTitleEdit(env *Env, opts titleEditOptions, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	if opts.title == "" {
		if opts.nonInteractive {
			env.err.Println("No title given. Use -m or -F option to specify a title. Aborting.")
			return nil
		}
		opts.title, err = input.BugTitleEditorInput(env.repo, snap.Title)
		if err == input.ErrEmptyTitle {
			env.out.Println("Empty title, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	if opts.title == snap.Title {
		env.err.Println("No change, aborting.")
	}

	_, err = b.SetTitle(text.CleanupOneLine(opts.title))
	if err != nil {
		return err
	}

	return b.Commit()
}
