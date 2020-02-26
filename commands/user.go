package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

var (
	userFieldsQuery string
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
		id, err = backend.ResolveIdentityPrefix(args[0])
	} else {
		id, err = backend.GetUserIdentity()
	}

	if err != nil {
		return err
	}

	if userFieldsQuery != "" {
		switch userFieldsQuery {
		case "email":
			fmt.Printf("%s\n", id.Email())
		case "login":
			fmt.Printf("%s\n", id.Login())
		case "humanId":
			fmt.Printf("%s\n", id.Id().Human())
		case "id":
			fmt.Printf("%s\n", id.Id())
		case "lastModification":
			fmt.Printf("%s\n", id.LastModification().
				Time().Format("Mon Jan 2 15:04:05 2006 +0200"))
		case "lastModificationLamport":
			fmt.Printf("%d\n", id.LastModificationLamport())
		case "metadata":
			for key, value := range id.ImmutableMetadata() {
				fmt.Printf("%s\n%s\n", key, value)
			}
		case "name":
			fmt.Printf("%s\n", id.Name())

		default:
			return fmt.Errorf("\nUnsupported field: %s\n", userFieldsQuery)
		}

		return nil
	}

	fmt.Printf("Id: %s\n", id.Id())
	fmt.Printf("Name: %s\n", id.Name())
	fmt.Printf("Email: %s\n", id.Email())
	fmt.Printf("Login: %s\n", id.Login())
	fmt.Printf("Last modification: %s (lamport %d)\n",
		id.LastModification().Time().Format("Mon Jan 2 15:04:05 2006 +0200"),
		id.LastModificationLamport())
	fmt.Println("Metadata:")
	for key, value := range id.ImmutableMetadata() {
		fmt.Printf("    %s --> %s\n", key, value)
	}
	// fmt.Printf("Protected: %v\n", id.IsProtected())

	return nil
}

var userCmd = &cobra.Command{
	Use:     "user [<user-id>]",
	Short:   "Display or change the user identity.",
	PreRunE: loadRepoEnsureUser,
	RunE:    runUser,
}

func init() {
	RootCmd.AddCommand(userCmd)
	userCmd.Flags().SortFlags = false

	userCmd.Flags().StringVarP(&userFieldsQuery, "field", "f", "",
		"Select field to display. Valid values are [email,humanId,id,lastModification,lastModificationLamport,login,metadata,name]")
}
