## git-bug bridge configure

Configure a new bridge.

### Synopsis

	Configure a new bridge by passing flags or/and using interactive terminal prompts. You can avoid all the terminal prompts by passing all the necessary flags to configure your bridge.
	Repository configuration can be made by passing either the --url flag or the --project and --owner flags. If the three flags are provided git-bug will use --project and --owner flags.
	Token configuration can be directly passed with the --token flag or in the terminal prompt. If you don't already have one you can use the interactive procedure to generate one.

```
git-bug bridge configure [flags]
```

### Examples

```
# Interactive example
[1]: github
[2]: launchpad-preview
target: 1
name [default]: default

Detected projects:
[1]: github.com/a-hilaly/git-bug
[2]: github.com/MichaelMure/git-bug

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

# For Github
git bug bridge configure \
    --name=default \
    --target=github \
    --owner=$(OWNER) \
    --project=$(PROJECT) \
    --token=$(TOKEN)

# For Launchpad
git bug bridge configure \
    --name=default \
    --target=launchpad-preview \
    --url=https://bugs.launchpad.net/ubuntu/
```

### Options

```
  -n, --name string      A distinctive name to identify the bridge
  -t, --target string    The target of the bridge. Valid values are [github,gitlab,launchpad-preview]
  -u, --url string       The URL of the target repository
  -o, --owner string     The owner of the target repository
  -T, --token string     The authentication token for the API
      --token-stdin      Will read the token from stdin and ignore --token
  -p, --project string   The name of the target repository
  -h, --help             help for configure
```

### SEE ALSO

* [git-bug bridge](git-bug_bridge.md)	 - Configure and use bridges to other bug trackers.

