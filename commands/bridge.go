package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
)

func newBridgeCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "bridge",
		Short:   "Configure and use bridges to other bug trackers.",
		PreRunE: loadBackend(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridge(env)
		}),
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newBridgeAuthCommand())
	cmd.AddCommand(newBridgeConfigureCommand())
	cmd.AddCommand(newBridgePullCommand())
	cmd.AddCommand(newBridgePushCommand())
	cmd.AddCommand(newBridgeRm())

	return cmd
}

func runBridge(env *Env) error {
	configured, err := bridge.ConfiguredBridges(env.backend)
	if err != nil {
		return err
	}

	for _, c := range configured {
		env.out.Println(c)
	}

	return nil
}
