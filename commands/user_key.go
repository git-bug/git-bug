package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
)

func newUserKeyCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "key [<user-id>]",
		Short:    "Display, add or remove keys to/from a user.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserKey(env, args)
		},
	}

	cmd.AddCommand(newUserKeyAddCommand())
	cmd.AddCommand(newUserKeyRmCommand())

	return cmd
}

func ResolveUser(repo *cache.RepoCache, args []string) (*cache.IdentityCache, []string, error) {
	var err error
	var id *cache.IdentityCache
	if len(args) > 0 {
		id, err = repo.ResolveIdentityPrefix(args[0])
		args = args[1:]
	} else {
		id, err = repo.GetUserIdentity()
	}
	return id, args, err
}

func runUserKey(env *Env, args []string) error {
	id, args, err := ResolveUser(env.backend, args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", args)
	}

	for _, key := range id.Keys() {
		fmt.Println(key.Fingerprint())
	}

	return nil
}
