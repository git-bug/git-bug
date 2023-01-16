package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBugStatusCloseCommand() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "close [BUG_ID]",
		Short:   "Mark a bug as closed",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugStatusClose(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	return cmd
}

func runBugStatusClose(env *execenv.Env, args []string) error {
	b, args, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	_, err = b.Close()
	if err != nil {
		return err
	}

	return b.Commit()
}
