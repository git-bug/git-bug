package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/spf13/cobra"
)

func runId(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return errors.New("only one identity can be displayed at a time")
	}

	var id *identity.Identity
	var err error

	if len(args) == 1 {
		id, err = identity.ReadLocal(repo, args[0])
	} else {
		id, err = identity.GetUserIdentity(repo)
	}

	if err != nil {
		return err
	}

	fmt.Printf("Id: %s\n", id.Id())
	fmt.Printf("Identity: %s\n", id.DisplayName())
	fmt.Printf("Name: %s\n", id.Name())
	fmt.Printf("Login: %s\n", id.Login())
	fmt.Printf("Email: %s\n", id.Email())
	fmt.Printf("Protected: %v\n", id.IsProtected())

	return nil
}

var idCmd = &cobra.Command{
	Use:     "id [<id>]",
	Short:   "Display or change the user identity",
	PreRunE: loadRepo,
	RunE:    runId,
}

func init() {
	RootCmd.AddCommand(idCmd)
	selectCmd.Flags().SortFlags = false
}
