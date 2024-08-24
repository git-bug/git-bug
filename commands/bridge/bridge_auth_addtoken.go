package bridgecmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge"
	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
)

type bridgeAuthAddTokenOptions struct {
	target string
	login  string
	user   string
}

func newBridgeAuthAddTokenCommand(env *execenv.Env) *cobra.Command {
	options := bridgeAuthAddTokenOptions{}

	cmd := &cobra.Command{
		Use:     "add-token [TOKEN]",
		Short:   "Store a new token",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgeAuthAddToken(env, options, args)
		}),
		Args: cobra.MaximumNArgs(1),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.target, "target", "t", "",
		fmt.Sprintf("The target of the bridge. Valid values are [%s]", strings.Join(bridge.Targets(), ",")))
	cmd.RegisterFlagCompletionFunc("target", completion.From(bridge.Targets()))
	flags.StringVarP(&options.login,
		"login", "l", "", "The login in the remote bug-tracker")
	flags.StringVarP(&options.user,
		"user", "u", "", "The user to add the token to. Default is the current user")
	cmd.RegisterFlagCompletionFunc("user", completion.User(env))

	return cmd
}

func runBridgeAuthAddToken(env *execenv.Env, opts bridgeAuthAddTokenOptions, args []string) error {
	// Note: as bridgeAuthAddTokenLogin is not checked against the remote bug-tracker,
	// it's possible to register a credential with an incorrect login (including bad case).
	// The consequence is that it will not get picked later by the bridge. I find that
	// checking it would require a cumbersome UX (need to provide a base URL for some bridges, ...)
	// so it's probably not worth it, unless we refactor that entirely.

	if opts.target == "" {
		return fmt.Errorf("flag --target is required")
	}
	if opts.login == "" {
		return fmt.Errorf("flag --login is required")
	}

	if !core.TargetExist(opts.target) {
		return fmt.Errorf("unknown target")
	}

	var value string

	if len(args) == 1 {
		value = args[0]
	} else {
		// Read from Stdin
		if isatty.IsTerminal(os.Stdin.Fd()) {
			env.Err.Println("Enter the token:")
		}
		reader := bufio.NewReader(os.Stdin)
		raw, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading from stdin: %v", err)
		}
		value = strings.TrimSuffix(raw, "\n")
	}

	var user *cache.IdentityCache
	var err error

	if opts.user == "" {
		user, err = env.Backend.GetUserIdentity()
	} else {
		user, err = env.Backend.Identities().ResolvePrefix(opts.user)
	}
	if err != nil {
		return err
	}

	metaKey, _ := bridge.LoginMetaKey(opts.target)
	login, ok := user.ImmutableMetadata()[metaKey]

	switch {
	case ok && login == opts.login:
		// nothing to do
	case ok && login != opts.login:
		return fmt.Errorf("this user is already tagged with a different %s login", opts.target)
	default:
		user.SetMetadata(metaKey, opts.login)
		err = user.Commit()
		if err != nil {
			return err
		}
	}

	token := auth.NewToken(opts.target, value)
	token.SetMetadata(auth.MetaKeyLogin, opts.login)

	if err := token.Validate(); err != nil {
		return errors.Wrap(err, "invalid token")
	}

	err = auth.Store(env.Repo, token)
	if err != nil {
		return err
	}

	env.Out.Printf("token %s added\n", token.ID())
	return nil
}
