package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newBridgeCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "bridge",
		Short:   "Configure and use bridges to other bug trackers.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridge(env)
		},
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
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	configured, err := bridge.ConfiguredBridges(backend)
	if err != nil {
		return err
	}

	for _, c := range configured {
		env.out.Println(c)
	}

	return nil
}
