package usercmd

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
)

func newUserAdoptCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "adopt USER_ID",
		Short:   "Adopt an existing identity as your own",
		Args:    cobra.ExactArgs(1),
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runUserAdopt(env, args)
		}),
		ValidArgsFunction: completion.User(env),
	}

	return cmd
}

func runUserAdopt(env *execenv.Env, args []string) error {
	prefix := args[0]

	i, err := env.Backend.Identities().ResolvePrefix(prefix)
	if err != nil {
		return err
	}

	err = env.Backend.SetUserIdentity(i)
	if err != nil {
		return err
	}

	env.Out.Printf("Your identity is now: %s\n", i.DisplayName())

	return nil
}
