package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/input"
)

type createUserOptions struct {
	name           string
	email          string
	avatarURL      string
	nonInteractive bool
}

func newUserCreateCommand() *cobra.Command {
	env := newEnv()

	options := createUserOptions{}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new identity.",
		PreRunE: loadBackend(env),
		RunE: closeBackend(env, func(cmd *cobra.Command, args []string) error {
			return runUserCreate(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.name, "name", "n", "", "Name to identify the user")
	flags.StringVarP(&options.email, "email", "e", "", "Email of the user")
	flags.StringVarP(&options.avatarURL, "avatar", "a", "", "Avatar URL")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runUserCreate(env *Env, opts createUserOptions) error {

	if !opts.nonInteractive && opts.name == "" {
		preName, err := env.backend.GetUserName()
		if err != nil {
			return err
		}
		opts.name, err = input.PromptDefault("Name", "name", preName, input.Required)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.email == "" {
		preEmail, err := env.backend.GetUserEmail()
		if err != nil {
			return err
		}

		opts.email, err = input.PromptDefault("Email", "email", preEmail, input.Required)
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.avatarURL == "" {
		var err error
		opts.avatarURL, err = input.Prompt("Avatar URL", "avatar")
		if err != nil {
			return err
		}
	}

	id, err := env.backend.NewIdentityRaw(opts.name, opts.email, "", opts.avatarURL, nil, nil)
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
