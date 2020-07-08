package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/select"
)

func newStatusOpenCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "open [<id>]",
		Short:    "Mark a bug as open.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusOpen(env, args)
		},
	}

	return cmd
}

func runStatusOpen(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	_, err = b.Open()
	if err != nil {
		return err
	}

	return b.Commit()
}
