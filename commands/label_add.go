package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newLabelAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "add [<id>] <label>[...]",
		Short:   "Add a label to a bug.",
		PreRunE: loadRepoEnsureUser(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(env, args)
		},
	}

	return cmd
}

func runLabelAdd(env *Env, args []string) error {
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

	changes, _, err := b.ChangeLabels(args, nil)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
