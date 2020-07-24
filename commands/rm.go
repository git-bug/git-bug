package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

func newRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm <id>",
		Short:    "Remove an existing bug.",
		Long:     "Remove an existing bug in the local repository. If the bug was imported from a bridge, specify the remote name to remove it from. Note removing bugs that were imported from bridges will not remove the bug remote, and will only remove the local copy of the bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(env, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	return cmd
}

func runRm(env *Env, args []string) (err error) {
	if len(args) == 0 {
		return errors.New("you must provide a bug prefix to remove")
	}

	err = env.backend.RemoveBug(args[0])

	if err != nil {
		return
	}

	env.out.Printf("bug %s removed\n", args[0])

	return
}
