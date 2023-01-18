package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/execenv"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/entities/bug"
)

func newBugDeselectCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deselect",
		Short: "Clear the implicitly selected bug",
		Example: `git bug select 2f15
git bug comment
git bug status
git bug deselect
`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugDeselect(env)
		}),
	}

	return cmd
}

func runBugDeselect(env *execenv.Env) error {
	err := _select.Clear(env.Backend, bug.Namespace)
	if err != nil {
		return err
	}

	return nil
}
