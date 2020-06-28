package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newStatusCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "status [<id>]",
		Short:   "Display or change a bug status.",
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(env, args)
		},
	}

	cmd.AddCommand(newStatusCloseCommand())
	cmd.AddCommand(newStatusOpenCommand())

	return cmd
}

func runStatus(env *Env, args []string) error {
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

	env.out.Println(snap.Status)

	return nil
}
