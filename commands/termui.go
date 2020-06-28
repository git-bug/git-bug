package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/termui"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newTermUICommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "termui",
		Aliases: []string{"tui"},
		Short:   "Launch the terminal UI.",
		PreRunE: loadRepoEnsureUser(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTermUI(env)
		},
	}

	return cmd
}

func runTermUI(env *Env) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	return termui.Run(backend)
}
