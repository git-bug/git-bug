package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/select"
)

func newStatusCloseCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "close [<id>]",
		Short:    "Mark a bug as closed.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusClose(env, args)
		},
	}

	return cmd
}

func runStatusClose(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	_, err = b.Close()
	if err != nil {
		return err
	}

	return b.Commit()
}
