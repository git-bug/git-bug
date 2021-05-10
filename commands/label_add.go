package commands

import (
	"github.com/spf13/cobra"

	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/text"
)

func newLabelAddCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:     "add [ID] LABEL...",
		Short:   "Add a label to a bug.",
		PreRunE: loadBackendEnsureUser(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(env, args)
		}),
	}

	return cmd
}

func runLabelAdd(env *Env, args []string) error {
	b, args, err := _select.ResolveBug(env.backend, args)
	if err != nil {
		return err
	}

	added := args

	changes, _, err := b.ChangeLabels(text.CleanupOneLineArray(added), nil)

	for _, change := range changes {
		env.out.Println(change)
	}

	if err != nil {
		return err
	}

	return b.Commit()
}
