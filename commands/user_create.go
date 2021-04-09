package commands

import (
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/input"
)

func newUserCreateCommand() *cobra.Command {
	env := newEnv()

	cmd := &cobra.Command{
		Use:      "create",
		Short:    "Create a new identity.",
		PreRunE:  loadBackend(env),
		PostRunE: closeBackend(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUserCreate(env)
		},
	}

	return cmd
}

func runUserCreate(env *Env) error {
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

	id, err := env.backend.NewIdentityRaw(name, email, "", avatarURL, nil, nil)
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
