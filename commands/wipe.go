package commands

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

func newWipeCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wipe",
		Short:   "Wipe git-bug from the git repository",
		PreRunE: execenv.LoadBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWipe(env)
		},
	}

	return cmd
}

func runWipe(env *execenv.Env) error {
	env.Out.Println("cleaning entities...")
	err := env.Backend.RemoveAll()
	if err != nil {
		_ = env.Backend.Close()
		return err
	}

	env.Out.Println("cleaning git config ...")
	err = env.Backend.ClearUserIdentity()
	if err != nil {
		_ = env.Backend.Close()
		return err
	}

	err = env.Backend.LocalConfig().RemoveAll("git-bug")
	if err != nil {
		_ = env.Backend.Close()
		return err
	}

	storage := env.Backend.LocalStorage()

	err = env.Backend.Close()
	if err != nil {
		return err
	}

	env.Out.Println("cleaning caches ...")
	err = storage.RemoveAll(".")
	if err != nil {
		return err
	}

	return nil
}
