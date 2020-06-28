package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newTitleCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "title [<id>]",
		Short:   "Display or change a title of a bug.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTitle(env, args)
		},
	}

	cmd.AddCommand(newTitleEditCommand())

	return cmd
}

func runTitle(env *Env, args []string) error {
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

	snap := b.Snapshot()

	env.out.Println(snap.Title)

	return nil
}
