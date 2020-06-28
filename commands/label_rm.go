package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/select"
)

func newLabelRmCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "rm [<id>] <label>[...]",
		Short:    "Remove a label from a bug.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelRm(env, args)
		},
	}

	return cmd
}

func runLabelRm(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	changes, _, err := b.ChangeLabels(nil, args)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
