package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

type rmOptions struct {
}

func newRmCommand() *cobra.Command {
	env := newEnv()
	options := rmOptions{}

	cmd := &cobra.Command{
		Use:      "rm <id>",
		Short:    "Remove an existing bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	return cmd
}

func runRm(env *Env, opts rmOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("you must provide a bug id prefix to remove")
	}

	err := env.backend.RemoveBug(args[0])
	if err != nil {
		return err
	}

	env.out.Printf("bug %s removed\n", args[0])

	return nil
}
