package bridgecmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/bridge"
	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/commands/completion"
	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/repository"
)

type bridgeNewOptions struct {
	name           string
	target         string
	params         core.BridgeParams
	token          string
	tokenStdin     bool
	nonInteractive bool
}

func newBridgeNewCommand(env *execenv.Env) *cobra.Command {
	options := bridgeNewOptions{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Configure a new bridge",
		Long: `Configure a new bridge by passing flags or/and using interactive terminal prompts.
You can avoid all the terminal prompts by passing all the necessary flags to configure your bridge.

Authentication tokens and required permissions by target:

Gitea:
  Requires a personal access token created at $BASE_URL/user/settings/applications
  (e.g. https://codeberg.org/user/settings/applications; works with any Gitea
  or Forgejo instance).
  With fine-grained scopes the minimum set is:
    - 'write:issue' (push issues, comments and labels; 'read:issue' is
      enough for pull only)
    - 'read:repository' (repository access check)
    - 'read:user' (identify the token's owner account)
  On instances without fine-grained scopes, use a token with write access
  to the repository, or a read-only token for pull only.

GitHub:
  Requires a classic personal access token (https://github.com/settings/tokens;
  fine-grained tokens starting with 'github_pat_' are not supported).
  Required scopes:
    - 'public_repo' for public repositories
    - 'repo' for private repositories
  No other scope (code, branches, pull request) is needed or used.

GitLab:
  Requires a personal access token created on the target GitLab instance
  ($BASE_URL/-/user_settings/personal_access_tokens).
  Required scopes:
    - 'api' (read/write) to import issues and push changes (issues,
      comments, labels, status). 'read_api' is enough if you only pull.
    - Read access to the project (at least 'Reporter' role, or public
      visibility) is required, and at least 'Developer' role to push.

Jira:
  Authenticates with your user account, so the account itself must have
  permission to view the project and to create, comment on, edit and
  transition issues.
    - Jira Cloud: an API token generated at
      https://id.atlassian.com/manage-profile/security/api-tokens
      (sent along with your username).
    - Jira Data Center / Server: a username and password (session
      authentication), or an API token.

Launchpad:
  No token required: the Launchpad bridge is read-only and currently uses
  the public Launchpad API without any authentication.`,
		Example: `# Interactive example
[1]: gitea
[2]: github
[3]: gitlab
[4]: jira
[5]: launchpad-preview

target: 2
name [default]: default

Detected projects:
[1]: github.com/git-bug/git-bug

[0]: Another project

Select option: 1

[1]: user provided token
[2]: interactive token creation
Select option: 1

You can generate a new token by visiting https://github.com/settings/tokens.
Choose 'Generate new token' and set the necessary access scope for your repository.

The access scope depend on the type of repository.
Public:
	- 'public_repo': to be able to read public repositories
Private:
	- 'repo'       : to be able to read private repositories

Enter token: 87cf5c03b64029f18ea5f9ca5679daa08ccbd700
Successfully configured bridge: default

# For Gitea
git bug bridge new \
    --name=default \
    --target=gitea \
    --url=https://codeberg.org/example-owner/example-repo \
    --token=$TOKEN

# For GitHub
git bug bridge new \
    --name=default \
    --target=github \
    --owner=example-owner \
    --project=example-repo \
    --token=$TOKEN

# For GitLab
git bug bridge new \
    --name=default \
    --target=gitlab \
    --url=https://gitlab.example.com/example-org/example-repo \
    --token=$TOKEN

# For Jira
git bug bridge new \
    --name=default \
    --target=jira \
    --base-url=https://jira.example.com \
    --project=PROJ \
    --login=user@example.com \
    --token=$TOKEN

# For Launchpad
git bug bridge new \
    --name=default \
    --target=launchpad-preview \
    --url=https://bugs.launchpad.net/ubuntu/`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBridgeNew(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.name, "name", "n", "", "A distinctive name to identify the bridge")
	flags.StringVarP(&options.target, "target", "t", "",
		fmt.Sprintf("The target of the bridge. Valid values are [%s]", strings.Join(bridge.Targets(), ",")))
	cmd.RegisterFlagCompletionFunc("target", completion.From(bridge.Targets()))
	flags.StringVarP(&options.params.URL, "url", "u", "", "The URL of the remote repository")
	flags.StringVarP(&options.params.BaseURL, "base-url", "b", "", "The base URL of your remote issue tracker")
	flags.StringVarP(&options.params.Login, "login", "l", "", "The login on your remote issue tracker")
	flags.StringVarP(&options.params.CredPrefix, "credential", "c", "", "The identifier or prefix of an already known credential for your remote issue tracker (see \"git-bug bridge auth\")")
	flags.StringVar(&options.token, "token", "", "A raw authentication token for the remote issue tracker")
	flags.BoolVar(&options.tokenStdin, "token-stdin", false, "Will read the token from stdin and ignore --token")
	flags.StringVarP(&options.params.Owner, "owner", "o", "", "The owner of the remote repository")
	flags.StringVarP(&options.params.Project, "project", "p", "", "The name of the remote repository")
	flags.BoolVar(&options.nonInteractive, "non-interactive", false, "Do not ask for user input")

	return cmd
}

func runBridgeNew(env *execenv.Env, opts bridgeNewOptions) error {
	var err error

	if (opts.tokenStdin || opts.token != "" || opts.params.CredPrefix != "") &&
		(opts.name == "" || opts.target == "") {
		return fmt.Errorf("you must provide a bridge name and target to configure a bridge with a credential")
	}

	// early fail
	if opts.params.CredPrefix != "" {
		if _, err := auth.LoadWithPrefix(env.Repo, opts.params.CredPrefix); err != nil {
			return err
		}
	}

	switch {
	case opts.tokenStdin:
		reader := bufio.NewReader(os.Stdin)
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading from stdin: %v", err)
		}
		opts.params.TokenRaw = strings.TrimSpace(token)
	case opts.token != "":
		opts.params.TokenRaw = opts.token
	}

	if !opts.nonInteractive && opts.target == "" {
		opts.target, err = promptTarget()
		if err != nil {
			return err
		}
	}

	if !opts.nonInteractive && opts.name == "" {
		opts.name, err = promptName(env.Repo)
		if err != nil {
			return err
		}
	}

	b, err := bridge.NewBridge(env.Backend, opts.target, opts.name)
	if err != nil {
		return err
	}

	err = b.Configure(opts.params, !opts.nonInteractive)
	if err != nil {
		return err
	}

	env.Out.Printf("Successfully configured bridge: %s\n", opts.name)
	return nil
}

func promptTarget() (string, error) {
	// TODO: use the reusable prompt from the input package
	targets := bridge.Targets()

	for {
		for i, target := range targets {
			fmt.Printf("[%d]: %s\n", i+1, target)
		}
		fmt.Printf("target: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')

		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		index, err := strconv.Atoi(line)
		if err != nil || index <= 0 || index > len(targets) {
			fmt.Println("invalid input")
			continue
		}

		return targets[index-1], nil
	}
}

func promptName(repo repository.RepoConfig) (string, error) {
	// TODO: use the reusable prompt from the input package
	const defaultName = "default"

	defaultExist := core.BridgeExist(repo, defaultName)

	for {
		if defaultExist {
			fmt.Printf("name: ")
		} else {
			fmt.Printf("name [%s]: ", defaultName)
		}

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		name := line
		if defaultExist && name == "" {
			continue
		}

		if name == "" {
			name = defaultName
		}

		if !core.BridgeExist(repo, name) {
			return name, nil
		}

		fmt.Println("a bridge with the same name already exist")
	}
}
