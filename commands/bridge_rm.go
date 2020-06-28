package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newBridgeRm() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "rm <name>",
		Short:   "Delete a configured bridge.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBridgeRm(env, args)
		},
		Args: cobra.ExactArgs(1),
	}

	return cmd
}

func runBridgeRm(env *Env, args []string) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	err = bridge.RemoveBridge(backend, args[0])
	if err != nil {
		return err
	}

	env.out.Printf("Successfully removed bridge configuration %v\n", args[0])
	return nil
}
