package commands

import (
	"github.com/spf13/cobra"
)

func newLabelLsCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List valid labels.",
		Long: `List valid labels.

Note: in the future, a proper label policy could be implemented where valid labels are defined in a configuration file. Until that, the default behavior is to return the list of labels already used.`,
		PreRunE: loadBackend(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runLabelLs(env)
		}),
	}

	return cmd
}

func runLabelLs(env *Env) error {
	labels := env.backend.ValidLabels()

	for _, l := range labels {
		env.out.Println(l)
	}

	return nil
}
