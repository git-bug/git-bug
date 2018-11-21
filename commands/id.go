package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/spf13/cobra"
)

func runId(cmd *cobra.Command, args []string) error {
	id, err := identity.GetIdentity(repo)
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
	Use:     "id",
	Short:   "Display or change the user identity",
	PreRunE: loadRepo,
	RunE:    runId,
}

func init() {
	RootCmd.AddCommand(idCmd)
}
