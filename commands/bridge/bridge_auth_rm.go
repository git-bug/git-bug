package bridgecmd

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
)

func newBridgeAuthRm(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm BRIDGE_ID",
		Short:   "Remove a credential",
		PreRunE: execenv.LoadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeAuthRm(env, args)
		},
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.BridgeAuth(env),
	}

	return cmd
}

func runBridgeAuthRm(env *execenv.Env, args []string) error {
	cred, err := auth.LoadWithPrefix(env.Repo, args[0])
	if err != nil {
		return err
	}

	err = auth.Remove(env.Repo, cred.ID())
	if err != nil {
		return err
	}

	env.Out.Printf("credential %s removed\n", cred.ID())
	return nil
}
