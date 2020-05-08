package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/MichaelMure/git-bug/validate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func runValidate(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	validator, err := validate.NewValidator(backend)
	if err != nil {
		return err
	}

	fmt.Printf("first commit signed with key: %s\n", identity.encodeKeyFingerprint(validator.FirstKey.publicKey.Fingerprint))

	var refErr error
	for _, ref := range args {
		hash, err := backend.ResolveRef(ref)
		if err != nil {
			return err
		}

		_, err = validator.ValidateRef(hash)
		if err != nil {
			refErr = errors.Wrapf(refErr, "ref %s check fail", ref)
			fmt.Printf("ref %s\tFAIL\n", ref)
		} else {
			fmt.Printf("ref %s\tOK\n", ref)
		}
	}
	if refErr != nil {
		return refErr
	}

	return nil
}

var validateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "Validate identities and commits signatures.",
	PreRunE: loadRepo,
	RunE:    runValidate,
}

func init() {
	RootCmd.AddCommand(validateCmd)
	validateCmd.Flags().SortFlags = false
}
