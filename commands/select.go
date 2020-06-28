package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func newSelectCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:   "select <id>",
		Short: "Select a bug for implicit use in future commands.",
		Example: `git bug select 2f15
git bug comment
git bug status
`,
		Long: `Select a bug for implicit use in future commands.

This command allows you to omit any bug <id> argument, for example:
  git bug show
instead of
  git bug show 2f153ca

The complementary command is "git bug deselect" performing the opposite operation.
`,
		PreRunE: loadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelect(env, args)
		},
	}

	return cmd
}

func runSelect(env *Env, args []string) error {
	if len(args) == 0 {
		return errors.New("You must provide a bug id")
	}

	backend, err := cache.NewRepoCache(env.repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	prefix := args[0]

	b, err := backend.ResolveBugPrefix(prefix)
	if err != nil {
		return err
	}

	err = _select.Select(backend, b.Id())
	if err != nil {
		return err
	}

	env.out.Printf("selected bug %s: %s\n", b.Id().Human(), b.Snapshot().Title)

	return nil
}
