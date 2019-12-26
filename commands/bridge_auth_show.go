package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

func runBridgeAuthShow(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	cred, err := auth.LoadWithPrefix(repo, args[0])
	if err != nil {
		return err
	}

	var userFmt string

	switch cred.UserId() {
	case auth.DefaultUserId:
		userFmt = colors.Red("default user")
	default:
		user, err := backend.ResolveIdentity(cred.UserId())
		if err != nil {
			return err
		}
		userFmt = user.DisplayName()

		defaultUser, _ := backend.GetUserIdentity()
		if cred.UserId() == defaultUser.Id() {
			userFmt = colors.Red(userFmt)
		}
	}

	fmt.Printf("Id: %s\n", cred.ID())
	fmt.Printf("Target: %s\n", cred.Target())
	fmt.Printf("Kind: %s\n", cred.Kind())
	fmt.Printf("User: %s\n", userFmt)
	fmt.Printf("Creation: %s\n", cred.CreateTime().Format(time.RFC822))

	switch cred := cred.(type) {
	case *auth.Token:
		fmt.Printf("Value: %s\n", cred.Value)
	}

	return nil
}

var bridgeAuthShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Display an authentication credential.",
	PreRunE: loadRepo,
	RunE:    runBridgeAuthShow,
	Args:    cobra.ExactArgs(1),
}

func init() {
	bridgeAuthCmd.AddCommand(bridgeAuthShowCmd)
}
