package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newBugAssigneeCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "assignee [BUG_ID]",
		Short:   "Display the assignee of a bug",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugAssignee(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	cmd.AddCommand(newBugAssigneeSetCommand(env))
	cmd.AddCommand(newBugAssigneeClearCommand(env))

	return cmd
}

func runBugAssignee(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	snap := b.Snapshot()

	if snap.Assignee == nil {
		env.Out.Println("(unassigned)")
	} else {
		env.Out.Println(snap.Assignee.DisplayName())
	}

	return nil
}
