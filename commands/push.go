package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newPushCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "push [<remote>]",
		Short:   "Push bugs update to a git remote.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(env, args)
		},
	}

	return cmd
}

func runPush(env *Env, args []string) error {
	if len(args) > 1 {
		return errors.New("Only pushing to one remote at a time is supported")
	}

	remote := "origin"
	if len(args) == 1 {
		remote = args[0]
	}

	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	stdout, err := backend.Push(remote)
	if err != nil {
		return err
	}

	env.out.Println(stdout)

	return nil
}
