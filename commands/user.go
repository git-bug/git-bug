package commands

import (
	"errors"
	"fmt"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

func runUser(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if len(args) > 1 {
		return errors.New("only one identity can be displayed at a time")
	}

	var id *cache.IdentityCache
	if len(args) == 1 {
		// TODO
		return errors.New("this is not working yet, cache need to be hacked on")
		id, err = backend.ResolveIdentityPrefix(args[0])
	} else {
		id, err = backend.GetUserIdentity()
	}

	if err != nil {
		return err
	}

	fmt.Printf("Id: %s\n", id.Id())
	fmt.Printf("Name: %s\n", id.Name())
	fmt.Printf("Login: %s\n", id.Login())
	fmt.Printf("Email: %s\n", id.Email())
	fmt.Printf("Protected: %v\n", id.IsProtected())

	return nil
}

var userCmd = &cobra.Command{
	Use:     "user [<id>]",
	Short:   "Display or change the user identity",
	PreRunE: loadRepo,
	RunE:    runUser,
}

func init() {
	RootCmd.AddCommand(userCmd)
	userCmd.Flags().SortFlags = false
}
