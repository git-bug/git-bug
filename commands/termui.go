package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/termui"
)

func newTermUICommand() *cobra.Command {
	env := execenv.NewEnv()

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
