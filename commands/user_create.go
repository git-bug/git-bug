package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/input"
)

type userCreateOptions struct {
	ArmoredKeyFile string
}

func newUserCreateCommand() *cobra.Command {
	env := newEnv()
	options := userCreateOptions{}

	cmd := &cobra.Command{
		Use:      "create",
		Short:    "Create a new identity.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserCreate(env, options)
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&options.ArmoredKeyFile, "key-file", "",
		"Take the armored PGP public key from the given file. Use - to read the message from the standard input",
	)

	return cmd
}

func runUserCreate(env *Env, opts userCreateOptions) error {
	preName, err := env.backend.GetUserName()
	if err != nil {
		return err
	}

	name, err := input.PromptDefault("Name", "name", preName, input.Required)
	if err != nil {
		return err
	}

	preEmail, err := env.backend.GetUserEmail()
	if err != nil {
		return err
	}

	email, err := input.PromptDefault("Email", "email", preEmail, input.Required)
	if err != nil {
		return err
	}

	avatarURL, err := input.Prompt("Avatar URL", "avatar")
	if err != nil {
		return err
	}

	var key *identity.Key
	if opts.ArmoredKeyFile != "" {
		armoredPubkey, err := input.TextFileInput(opts.ArmoredKeyFile)
		if err != nil {
			return err
		}

		key, err = identity.NewKeyFromArmored(armoredPubkey)
		if err != nil {
			return err
		}

		fmt.Printf("Using key from file `%s`:\n%s\n", opts.ArmoredKeyFile, armoredPubkey)
	}

	id, err := env.backend.NewIdentityWithKeyRaw(name, email, "", avatarURL, nil, key)
	if err != nil {
		return err
	}

	err = id.CommitAsNeeded()
	if err != nil {
		return err
	}

	set, err := env.backend.IsUserIdentitySet()
	if err != nil {
		return err
	}

	if !set {
		err = env.backend.SetUserIdentity(id)
		if err != nil {
			return err
		}
	}

	env.err.Println()
	env.out.Println(id.Id())

	return nil
}
