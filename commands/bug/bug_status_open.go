package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBugStatusOpenCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "open [BUG_ID]",
		Short:   "Mark a bug as open",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugStatusOpen(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	return cmd
}

func runBugStatusOpen(env *execenv.Env, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	_, err = b.Open()
	if err != nil {
		return err
	}

	return b.Commit()
}
