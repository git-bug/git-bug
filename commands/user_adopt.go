package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newUserAdoptCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "adopt <user-id>",
		Short:   "Adopt an existing identity as your own.",
		Args:    cobra.ExactArgs(1),
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserAdopt(env, args)
		},
	}

	return cmd
}

func runUserAdopt(env *Env, args []string) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	prefix := args[0]

	i, err := backend.ResolveIdentityPrefix(prefix)
	if err != nil {
		return err
	}

	err = backend.SetUserIdentity(i)
	if err != nil {
		return err
	}

	env.out.Printf("Your identity is now: %s\n", i.DisplayName())

	return nil
}
