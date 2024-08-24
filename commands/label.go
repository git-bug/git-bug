package commands

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newLabelCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "List valid labels",
		Long: `List valid labels.

Note: in the future, a proper label policy could be implemented where valid labels are defined in a configuration file. Until that, the default behavior is to return the list of labels already used.`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runLabel(env)
		}),
	}

	return cmd
}

func runLabel(env *execenv.Env) error {
	labels := env.Backend.Bugs().ValidLabels()

	for _, l := range labels {
		env.Out.Println(l)
	}

	return nil
}
