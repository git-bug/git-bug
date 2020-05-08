package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

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

func runKey(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	id, args, err := ResolveUser(backend, args)
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

var keyCmd = &cobra.Command{
	Use:     "key [<user-id>]",
	Short:   "Display, add or remove keys to/from a user.",
	PreRunE: loadRepoEnsureUser,
	RunE:    runKey,
}

func init() {
	userCmd.AddCommand(keyCmd)
}
