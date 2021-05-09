package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/select"
)

func newDeselectCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "deselect",
		Short: "Clear the implicitly selected bug.",
		Example: `git bug select 2f15
git bug comment
git bug status
git bug deselect
`,
		PreRunE: loadBackend(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runDeselect(env)
		}),
	}

	return cmd
}

func runDeselect(env *Env) error {
	err := _select.Clear(env.backend)
	if err != nil {
		return err
	}

	return nil
}
