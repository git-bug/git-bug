package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/select"
)

func newLabelAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "add [<id>] <label>[...]",
		Short:    "Add a label to a bug.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(env, args)
		},
	}

	return cmd
}

func runLabelAdd(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	changes, _, err := b.ChangeLabels(args, nil)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
