package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newStatusOpenCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "open [<id>]",
		Short:   "Mark a bug as open.",
		PreRunE: loadRepoEnsureUser(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusOpen(env, args)
		},
	}

	return cmd
}

func runStatusOpen(env *Env, args []string) error {
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

	_, err = b.Open()
	if err != nil {
		return err
	}

	return b.Commit()
}
