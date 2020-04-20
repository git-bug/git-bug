package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/input"
	"github.com/spf13/cobra"
)

type userKeyAddOptions struct {
	ArmoredFile string
	Armored     string
}

func newUserKeyAddCommand() *cobra.Command {
	env := newEnv()
	options := userKeyAddOptions{}

	cmd := &cobra.Command{
		Use:      "add [<user-id>]",
		Short:    "Add a PGP key from a user.",
		PreRunE:  loadBackendEnsureUser(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserKeyAdd(env, options, args)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.ArmoredFile, "file", "F", "",
		"Take the armored PGP public key from the given file. Use - to read the message from the standard input",
	)

	flags.StringVarP(&options.Armored, "key", "k", "",
		"Provide the armored PGP public key from the command line",
	)

	return cmd
}


func runUserKeyAdd(env *Env, opts userKeyAddOptions, args []string) error {
	id, args, err := ResolveUser(env.backend, args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", args)
	}

	if opts.ArmoredFile != "" && opts.Armored == "" {
		opts.Armored, err = input.TextFileInput(opts.ArmoredFile)
		if err != nil {
			return err
		}
	}

	if opts.ArmoredFile == "" && opts.Armored == "" {
		opts.Armored, err = input.IdentityVersionKeyEditorInput(env.backend, "")
		if err == input.ErrEmptyMessage {
			fmt.Println("Empty PGP key, aborting.")
			return nil
		}
		if err != nil {
			return err
		}
	}

	key, err := identity.NewKeyFromArmored(opts.Armored)
	if err != nil {
		return err
	}

	err = id.Mutate(identity.AddKeyMutator(key))
	if err != nil {
		return err
	}

	return id.Commit()
}


