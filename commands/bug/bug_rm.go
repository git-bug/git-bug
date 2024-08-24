package bugcmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newBugRmCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm BUG_ID",
		Short:   "Remove an existing bug",
		Long:    "Remove an existing bug in the local repository. Note removing bugs that were imported from bridges will not remove the bug on the remote, and will only remove the local copy of the bug.",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugRm(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	return cmd
}

func runBugRm(env *execenv.Env, args []string) (err error) {
	if len(args) == 0 {
		return errors.New("you must provide a bug prefix to remove")
	}

	err = env.Backend.Bugs().Remove(args[0])

	if err != nil {
		return
	}

	env.Out.Printf("bug %s removed\n", args[0])

	return
}
