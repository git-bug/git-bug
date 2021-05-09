package commands

import (
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/spf13/cobra"
)

func newStatusOpenCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "open [ID]",
		Short:   "Mark a bug as open.",
		PreRunE: loadBackendEnsureUser(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runStatusOpen(env, args)
		}),
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
