package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/util/text"
)

func newBugLabelNewCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [BUG_ID] LABEL...",
		Short:   "Add a label to a bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugLabelNew(env, args)
		}),
		ValidArgsFunction: BugAndLabelsCompletion(env, true),
	}

	return cmd
}

func runBugLabelNew(env *execenv.Env, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	added := args

	changes, _, err := b.ChangeLabels(text.CleanupOneLineArray(added), nil)

	for _, change := range changes {
		env.Out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
