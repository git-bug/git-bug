package bridgecmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newBridgeRm() *cobra.Command {
	env := execenv.NewEnv()

	cmd := &cobra.Command{
		Use:     "rm NAME",
		Short:   "Delete a configured bridge",
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgeRm(env, args)
		}),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.Bridge(env),
	}

	return cmd
}

func runBridgeRm(env *execenv.Env, args []string) error {
	err := bridge.RemoveBridge(env.Backend, args[0])
	if err != nil {
		return err
	}

	env.Out.Printf("Successfully removed bridge configuration %v\n", args[0])
	return nil
}
