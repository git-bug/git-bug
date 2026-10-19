package boardcmd

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/commands/input"
	"github.com/git-bug/git-bug/util/text"
)

type boardDescriptionEditOptions struct {
	description    string
	nonInteractive bool
}

func newBoardDescriptionEditCommand(env *execenv.Env) *cobra.Command {
	options := boardDescriptionEditOptions{}

	cmd := &cobra.Command{
		Use:     "edit [BUG_ID]",
		Short:   "Edit a description of a board",
		PreRunE: execenv.LoadBackend(env, execenv.EnsureUser()),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBoardDescriptionEdit(env, options, args)
		}),
		ValidArgsFunction: BoardCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.description, "description", "d", "",
		"Provide a description for the board",
	)
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBoardDescriptionEdit(env *execenv.Env, opts boardDescriptionEditOptions, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	if opts.description == "" {
		if opts.nonInteractive {
			env.Err.Println("No description given. Aborting.")
			return nil
		}
		opts.description, err = input.PromptDefault("Board description", "description", snap.Description, input.Required)
		if err != nil {
			return err
		}
	}

	if opts.description == snap.Description {
		env.Err.Println("No change, aborting.")
		return nil
	}

	_, err = b.SetDescription(text.CleanupOneLine(opts.description))
	if err != nil {
		return err
	}

	return b.Commit()
}
