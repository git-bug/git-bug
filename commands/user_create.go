package commands

import (
	"fmt"
	"os"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runUserCreate(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	preName, err := backend.GetUserName()
	if err != nil {
		return err
	}

	name, err := input.Prompt("Name", "name", preName, input.Required)
	if err != nil {
		return err
	}

	preEmail, err := backend.GetUserEmail()
	if err != nil {
		return err
	}

	email, err := input.Prompt("Email", "email", preEmail, input.Required)
	if err != nil {
		return err
	}

	login, err := input.Prompt("Avatar URL", "avatar", "")
	if err != nil {
		return err
	}

	id, err := backend.NewIdentityRaw(name, email, "", login, nil)
	if err != nil {
		return err
	}

	err = id.CommitAsNeeded()
	if err != nil {
		return err
	}

	set, err := backend.IsUserIdentitySet()
	if err != nil {
		return err
	}

	if !set {
		err = backend.SetUserIdentity(id)
		if err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintln(os.Stderr)
	fmt.Println(id.Id())

	return nil
}

var userCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new identity.",
	PreRunE: loadRepo,
	RunE:    runUserCreate,
}

func init() {
	userCmd.AddCommand(userCreateCmd)
	userCreateCmd.Flags().SortFlags = false
}
