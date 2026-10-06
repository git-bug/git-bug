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
		PreRunE: execenv.LoadBackend(env, execenv.EnsureUser(), execenv.FollowChanges()),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return termui.Run(cmd.Context(), env.Backend)
		}),
	}

	return cmd
}
