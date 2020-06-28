package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newLsLabelCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "ls-label",
		Short: "List valid labels.",
		Long: `List valid labels.

Note: in the future, a proper label policy could be implemented where valid labels are defined in a configuration file. Until that, the default behavior is to return the list of labels already used.`,
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLsLabel(env)
		},
	}

	return cmd
}

func runLsLabel(env *Env) error {
	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	labels := backend.ValidLabels()

	for _, l := range labels {
		env.out.Println(l)
	}

	return nil
}
