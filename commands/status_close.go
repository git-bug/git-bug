package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newStatusCloseCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "close [<id>]",
		Short:   "Mark a bug as closed.",
		PreRunE: loadRepoEnsureUser(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusClose(env, args)
		},
	}

	return cmd
}

func runStatusClose(env *Env, args []string) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	_, err = b.Close()
	if err != nil {
		return err
	}

	return b.Commit()
}
