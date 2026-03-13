package bugcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
)

func newBugAssigneeSetCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set [BUG_ID] USER_ID",
		Short:   "Assign a user to a bug",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugAssigneeSet(env, args)
		}),
		ValidArgsFunction: BugAndUserCompletion(env),
	}
	return cmd
}

func runBugAssigneeSet(env *execenv.Env, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return fmt.Errorf("missing user id argument")
	}
	user, err := env.Backend.Identities().ResolvePrefix(args[0])
	if err != nil {
		return err
	}
	_, err = b.SetAssignee(user.Id(), user)
	if err != nil {
		return err
	}
	return b.Commit()
}

func BugAndUserCompletion(env *execenv.Env) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return BugCompletion(env)(cmd, args, toComplete)
		}
		return completion.User(env)(cmd, args, toComplete)
	}
}
