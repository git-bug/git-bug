package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bridge"
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

var (
	bridgeAuthAddTokenTarget string
	bridgeAuthAddTokenLogin  string
	bridgeAuthAddTokenUser   string
)

func runBridgeTokenAdd(cmd *cobra.Command, args []string) error {
	if bridgeAuthAddTokenTarget == "" {
		return fmt.Errorf("flag --target is required")
	}
	if bridgeAuthAddTokenLogin == "" {
		return fmt.Errorf("flag --login is required")
	}

	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	if !core.TargetExist(bridgeAuthAddTokenTarget) {
		return fmt.Errorf("unknown target")
	}

	var value string

	if len(args) == 1 {
		value = args[0]
	} else {
		// Read from Stdin
		if isatty.IsTerminal(os.Stdin.Fd()) {
			fmt.Println("Enter the token:")
		}
		reader := bufio.NewReader(os.Stdin)
		raw, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading from stdin: %v", err)
		}
		value = strings.TrimSuffix(raw, "\n")
	}

	var user *cache.IdentityCache

	if bridgeAuthAddTokenUser == "" {
		user, err = backend.GetUserIdentity()
	} else {
		user, err = backend.ResolveIdentityPrefix(bridgeAuthAddTokenUser)
	}
	if err != nil {
		return err
	}

	metaKey, _ := bridge.LoginMetaKey(bridgeAuthAddTokenTarget)
	login, ok := user.ImmutableMetadata()[metaKey]

	switch {
	case ok && login == bridgeAuthAddTokenLogin:
		// nothing to do
	case ok && login != bridgeAuthAddTokenLogin:
		return fmt.Errorf("this user is already tagged with a different %s login", bridgeAuthAddTokenTarget)
	default:
		user.SetMetadata(metaKey, bridgeAuthAddTokenLogin)
		err = user.Commit()
		if err != nil {
			return err
		}
	}

	token := auth.NewToken(bridgeAuthAddTokenTarget, value)
	token.SetMetadata(auth.MetaKeyLogin, bridgeAuthAddTokenLogin)

	if err := token.Validate(); err != nil {
		return errors.Wrap(err, "invalid token")
	}

	err = auth.Store(repo, token)
	if err != nil {
		return err
	}

	fmt.Printf("token %s added\n", token.ID())
	return nil
}

var bridgeAuthAddTokenCmd = &cobra.Command{
	Use:     "add-token [<token>]",
	Short:   "Store a new token",
	PreRunE: loadRepoEnsureUser,
	RunE:    runBridgeTokenAdd,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	bridgeAuthCmd.AddCommand(bridgeAuthAddTokenCmd)
	bridgeAuthAddTokenCmd.Flags().StringVarP(&bridgeAuthAddTokenTarget, "target", "t", "",
		fmt.Sprintf("The target of the bridge. Valid values are [%s]", strings.Join(bridge.Targets(), ",")))
	bridgeAuthAddTokenCmd.Flags().StringVarP(&bridgeAuthAddTokenLogin,
		"login", "l", "", "The login in the remote bug-tracker")
	bridgeAuthAddTokenCmd.Flags().StringVarP(&bridgeAuthAddTokenUser,
		"user", "u", "", "The user to add the token to. Default is the current user")
	bridgeAuthAddTokenCmd.Flags().SortFlags = false
}
