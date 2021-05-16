## git-bug bridge new

Configure a new bridge

### Synopsis

Configure a new bridge by passing flags or/and using interactive terminal prompts. You can avoid all the terminal prompts by passing all the necessary flags to configure your bridge.

```
git-bug bridge new [flags]
```

### Examples

```
# Interactive example
[1]: github
[2]: gitlab
[3]: jira
[4]: launchpad-preview

target: 1
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

# For GitHub (classic personal access token, see https://github.com/settings/tokens;
#   fine-grained tokens starting with 'github_pat_' are not supported)
#   Required scopes:
#     - Read/write repository content ('repo' for private or 'public_repo' for
#       public repos) is needed to import issues and to edit, comment on and
#       push them back.
#
#   No other scope (code, branches, pull request) is needed or used.
git bug bridge new \
    --name=default \
    --target=github \
    --owner=example-owner \
    --project=example-repo \
    --token=$TOKEN

# For Launchpad
#   No token required: the Launchpad bridge is read-only and currently uses
#   the public Launchpad API without any authentication.
git bug bridge new \
    --name=default \
    --target=launchpad-preview \
    --url=https://bugs.launchpad.net/ubuntu/

# For Gitlab (personal access token, create one at
#   $BASE_URL/-/user_settings/personal_access_tokens, e.g.
#   https://gitlab.com/-/user_settings/personal_access_tokens)
#   Required scopes:
#     - 'api' (read/write) to import issues and push changes (issues,
#       comments, labels, status). 'read_api' is enough if you only pull.
#     - Read access to the project (at least 'Reporter' role, or the
#       project/repository visibility set to 'public') is required.
git bug bridge new \
    --name=default \
    --target=gitlab \
    --url=https://gitlab.example.com/example-org/example-repo \
    --token=$TOKEN

# For Jira
#   No project-specific scope is needed: the bridge authenticates with your
#   user account and only requires:
#     - Jira Cloud: an API token generated at
#       https://id.atlassian.com/manage-profile/security/api-tokens
#       (sent along with your username), and permission to view and edit
#       issues in the project.
#     - Jira Data Center / Server: a username and password (session
#       authentication), or an API token, with the same project permissions.
```

### Options

```
  -n, --name string         A distinctive name to identify the bridge
  -t, --target string       The target of the bridge. Valid values are [gitea-preview,github,gitlab,jira,launchpad-preview]
  -u, --url string          The URL of the remote repository
  -b, --base-url string     The base URL of your remote issue tracker
  -l, --login string        The login on your remote issue tracker
  -c, --credential string   The identifier or prefix of an already known credential for your remote issue tracker (see "git-bug bridge auth")
      --token string        A raw authentication token for the remote issue tracker
      --token-stdin         Will read the token from stdin and ignore --token
  -o, --owner string        The owner of the remote repository
  -p, --project string      The name of the remote repository
      --non-interactive     Do not ask for user input
  -h, --help                help for new
```

### SEE ALSO

* [git-bug bridge](git-bug_bridge.md)	 - List bridges to other bug trackers

