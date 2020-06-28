package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newLabelCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "label [<id>]",
		Short:   "Display, add or remove labels to/from a bug.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabel(env, args)
		},
	}

	cmd.AddCommand(newLabelAddCommand())
	cmd.AddCommand(newLabelRmCommand())

	return cmd
}

func runLabel(env *Env, args []string) error {
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

	for _, l := range snap.Labels {
		env.out.Println(l)
	}

	return nil
}
