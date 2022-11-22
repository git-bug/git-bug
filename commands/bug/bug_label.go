package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/bug/select"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBugLabelCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "label [BUG_ID]",
		Short:   "Display labels of a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugLabel(env, args)
		}),
		ValidArgsFunction: completion.Bug(env),
	}

	cmd.AddCommand(newBugLabelNewCommand())
	cmd.AddCommand(newBugLabelRmCommand())

	return cmd
}

func runBugLabel(env *execenv.Env, args []string) error {
	b, args, err := _select.ResolveBug(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	for _, l := range snap.Labels {
		env.Out.Println(l)
	}

	return nil
}
