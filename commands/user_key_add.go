package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/MichaelMure/git-bug/validate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	keyAddArmoredFile string
	keyAddArmored     string
)

func runKeyAdd(cmd *cobra.Command, args []string) error {
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

	if keyAddArmoredFile != "" && keyAddArmored == "" {
		keyAddArmored, err = input.TextFileInput(keyAddArmoredFile)
		if err != nil {
			return err
		}
	}

	if keyAddArmoredFile == "" && keyAddArmored == "" {
		keyAddArmored, err = input.IdentityVersionKeyEditorInput(backend, "")
		if err == input.ErrEmptyMessage {
			fmt.Println("Empty PGP key, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	key, err := identity.NewKeyFromArmored(keyAddArmored)
	if err != nil {
		return err
	}

	validator, err := validate.NewValidator(backend)
	if err != nil {
		return errors.Wrap(err, "failed to create validator")
	}
	commitHash := validator.KeyCommitHash(key.publicKey().KeyId)
	if commitHash != "" {
		return fmt.Errorf("key id %d is already used by the key introduced in commit %s", key.publicKey.KeyId, commitHash)
	}

	err = id.Mutate(func(mutator identity.Mutator) identity.Mutator {
		mutator.Keys = append(mutator.Keys, key)
		return mutator
	})
	if err != nil {
		return err
	}

	return id.Commit()
}

var keyAddCmd = &cobra.Command{
	Use:     "add [<user-id>]",
	Short:   "Add a PGP key from a user.",
	PreRunE: loadRepoEnsureUser,
	RunE:    runKeyAdd,
}

func init() {
	keyCmd.AddCommand(keyAddCmd)

	keyAddCmd.Flags().SortFlags = false

	keyAddCmd.Flags().StringVarP(&keyAddArmoredFile, "file", "F", "",
		"Take the armored PGP public key from the given file. Use - to read the message from the standard input",
	)

	keyAddCmd.Flags().StringVarP(&keyAddArmored, "key", "k", "",
		"Provide the armored PGP public key from the command line",
	)
}
