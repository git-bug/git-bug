package bridgecmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge"
	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
)

type bridgeAuthAddTokenOptions struct {
	target  string
	login   string
	user    string
	baseURL string
}

func newBridgeAuthAddTokenCommand(env *execenv.Env) *cobra.Command {
	options := bridgeAuthAddTokenOptions{}

	cmd := &cobra.Command{
		Use:     "add-token [TOKEN]",
		Short:   "Store a new token",
		PreRunE: execenv.LoadBackend(env, execenv.EnsureUser()),
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
	flags.StringVar(&options.baseURL, "base-url", "", "The server base URL for instance-scoped credentials")

	return cmd
}

func runBridgeAuthAddToken(env *execenv.Env, opts bridgeAuthAddTokenOptions, args []string) error {
	// The login is supplied by the user, not verified against the remote server.

	if opts.target == "" {
		return fmt.Errorf("flag --target is required")
	}
	if opts.login == "" {
		return fmt.Errorf("flag --login is required")
	}

	if !core.TargetExist(opts.target) {
		return fmt.Errorf("unknown target")
	}
	identityMetadata, credentialMetadata, err := bridge.CredentialMetadata(opts.target, opts.login, opts.baseURL)
	if err != nil {
		return err
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

	if opts.user == "" {
		user, err = env.Backend.GetUserIdentity(env.Ctx)
	} else {
		user, err = env.Backend.Identities().ResolvePrefix(env.Ctx, opts.user)
	}
	if err != nil {
		return err
	}

	for key, login := range identityMetadata {
		if existing, ok := user.ImmutableMetadata()[key]; ok && existing != login {
			return fmt.Errorf("this user is already tagged with a different %s login", opts.target)
		}
	}

	token := auth.NewToken(opts.target, value)
	token.SetMetadata(auth.MetaKeyLogin, opts.login)
	for key, value := range credentialMetadata {
		token.SetMetadata(key, value)
	}

	if err := token.Validate(); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	for key, login := range identityMetadata {
		if _, present := user.ImmutableMetadata()[key]; !present {
			user.SetMetadata(key, login)
		}
	}
	if err := user.CommitAsNeeded(); err != nil {
		return err
	}

	err = auth.Store(env.Repo, token)
	if err != nil {
		return err
	}

	env.Out.Printf("token %s added\n", token.ID())
	return nil
}
