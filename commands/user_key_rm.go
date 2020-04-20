package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runKeyRm(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if len(args) == 0 {
		return errors.New("missing key ID")
	}

	keyFingerprint := args[0]
	args = args[1:]

	id, args, err := ResolveUser(backend, args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", args)
	}

	fingerprint, err := identity.DecodeKeyFingerprint(keyFingerprint)
	if err != nil {
		return err
	}

	var removedKey *identity.Key
	err = id.Mutate(func(mutator identity.Mutator) identity.Mutator {
		for j, key := range mutator.Keys {
			pubkey, err := key.GetPublicKey()
			if err != nil {
				fmt.Printf("Warning: failed to decode public key: %s", err)
				continue
			}

			if pubkey.Fingerprint == fingerprint {
				removedKey = key
				copy(mutator.Keys[j:], mutator.Keys[j+1:])
				mutator.Keys = mutator.Keys[:len(mutator.Keys)-1]
				break
			}
		}
		return mutator
	})

	if err != nil {
		return err
	}

	if removedKey == nil {
		return errors.New("key not found")
	}

	return id.Commit()
}

var keyRmCmd = &cobra.Command{
	Use:     "rm <key-fingerprint> [<user-id>]",
	Short:   "Remove a PGP key from the adopted or the specified user.",
	PreRunE: loadRepoEnsureUser,
	RunE:    runKeyRm,
}

func init() {
	keyCmd.AddCommand(keyRmCmd)
}
