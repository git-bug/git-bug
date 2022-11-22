package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/bug/select"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/util/text"
)

func newBugLabelRmCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "rm [BUG_ID] LABEL...",
		Short:   "Remove a label from a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugLabelRm(env, args)
		}),
		ValidArgsFunction: completion.BugAndLabels(env, false),
	}

	return cmd
}

func runBugLabelRm(env *execenv.Env, args []string) error {
	b, args, err := _select.ResolveBug(env.Backend, args)
	if err != nil {
		return err
	}

	removed := args

	changes, _, err := b.ChangeLabels(nil, text.CleanupOneLineArray(removed))

	for _, change := range changes {
		env.Out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
