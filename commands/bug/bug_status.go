package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/bug/select"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBugStatusCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "status [BUG_ID]",
		Short:   "Display the status of a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugStatus(env, args)
		}),
		ValidArgsFunction: completion.Bug(env),
	}

	cmd.AddCommand(newBugStatusCloseCommand())
	cmd.AddCommand(newBugStatusOpenCommand())

	return cmd
}

func runBugStatus(env *execenv.Env, args []string) error {
	b, args, err := _select.ResolveBug(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	env.Out.Println(snap.Status)

	return nil
}
