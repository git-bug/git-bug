package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newBugAssigneeClearCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clear [BUG_ID]",
		Short:   "Remove the assignee from a bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugAssigneeClear(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}
	return cmd
}

func runBugAssigneeClear(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}
	_, err = b.Unassign()
	if err != nil {
		return err
	}
	return b.Commit()
}
