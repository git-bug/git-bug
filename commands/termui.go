package commands

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/termui"
)

func newTermUICommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "termui",
		Aliases: []string{"tui"},
		Short:   "Launch the terminal UI",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runTermUI(env)
		}),
	}

	return cmd
}

func runTermUI(env *execenv.Env) error {
	return termui.Run(env.Backend)
}
