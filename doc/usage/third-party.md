# Using third-party platforms via bridges<a name="using-bridges"></a>

This page provides an overview of how to use _bridges_ to sync issues to and
from third-party platforms.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Overview](#overview)
- [Supported bridges](#supported-bridges)
- [Getting started](#getting-started)
- [Tokens and required permissions](#tokens-and-required-permissions)
  - [GitHub](#github)
  - [GitLab](#gitlab)
  - [Jira](#jira)
  - [Launchpad](#launchpad)
- [Interacting with the bridge](#interacting-with-the-bridge)

<!-- mdformat-toc end -->

## Overview<a name="overview"></a>

Bridges within `git-bug` are bi-directional, incremental, and speedy gateways to
third-party platforms. Configuring a bridge allows you to push and pull issues
to and from a third party platform.

This expands the utility and function of `git-bug`: because issues are just
objects in your git repository, you can import issues from a bridge to work on
them in bulk, offline, in your preferred environment, at your own pace. When
you're ready to push your issues back to the external platform again, you'll be
able to synchronize the changes you made with one simple command.

<p align="center">
    <img src="../assets/bridge-workflow.png" alt="Bridge workflow">
</p>

This has several benefits:

- works offline, including edition
- browsing is pretty much instant
- you get to choose the UI you prefer between CLI, interactive TUI or in your
  browser with the WEBUI
- you have a near-complete archive of issues locally, embedded in your git
  repository, in case the external platform becomes inaccessible
- you are free to move to another platform -- your issues follow wherever your
  repo goes!

## Supported bridges<a name="supported-bridges"></a>

We support a number of bridges:

- Jira
- GitHub
- GitLab
- Launchpad
- Gitea

_For a full list of the features enabled for each bridge, see the
[feature matrix][docs/feature-matrix]._

## Getting started<a name="getting-started"></a>

1. From within a git repository, run `git bug bridge new` to start the
   configuration wizard
2. Choose the type of bridge you want to configure, e.g. `github`
3. Type a name for the bridge configuration. As you can configure multiple
   bridges, this name will allow you to choose when there is an ambiguity.
4. If you already have a repository created on the external platform, and your
   local git repository has that as a remote, the configuration wizard will
   automatically detect the URL. Otherwise, please ensure you enter the
   appropriate URL for the remote project: something like
   `https://github.com/git-bug/git-bug`
5. Create an access token. You can either use the interactive token creation,
   enter it on your own token, or use an existing token if you already have one.
   The [permissions required for each service](#tokens-and-required-permissions)
   differ, check the details for your platform below.

That's it! Once you've completed the wizard, you'll have successfully configured
a bridge.

## Tokens and required permissions<a name="tokens-and-required-permissions"></a>

Bridges authenticate against their platform with a token (except Launchpad,
which currently does not need one). Every bridge performs at least the following
API operations, which determine the minimum permissions a token needs:

- **pull**: read the issue list, issue bodies, comments, labels and
  status/title change events
- **push** (GitHub, GitLab and Jira only): create issues, edit title and body,
  add and edit comments, change labels and toggle the status (open/close)

### GitHub<a name="github"></a>

Tokens are created at <https://github.com/settings/tokens>, or via the
interactive OAuth device flow offered by the `git bug bridge new` wizard.

The bridge accepts classic personal access tokens (legacy 40-character tokens
and ones starting with `ghp_`, `gho_`, `ghu_`, `ghs_` or `ghr_`). Fine-grained
tokens (starting with `github_pat_`) are currently rejected when pasted
manually.

Required scopes (classic tokens): `public_repo` is enough for public
repositories; the full `repo` scope is required for private repositories.
These scopes grant read and write access to issues, comments and labels, which
is what the bridge needs in both directions. No other scope (code, branches,
pull requests) is used or required.

The interactive device flow creates a token with a single scope, chosen
according to the repository visibility: `public_repo` for public repositories
and `repo` for private ones.

### GitLab<a name="gitlab"></a>

Personal access tokens are created on the GitLab instance you want to bridge,
at `$BASE_URL/-/user_settings/personal_access_tokens` (specify the instance
with `--base-url`; defaults to `gitlab.com`).

- **Scopes**: the `api` scope (read and write) is required so the bridge can
  import issues and push changes back. If you only plan to pull issues, the
  `read_api` scope is enough.
- **Project role**: the token owner needs read access to the project to pull
  (any member role, or public project visibility), and at least the *Developer*
  role to push, since pushing creates issues, comments and label/status
  changes on your behalf.

### Jira<a name="jira"></a>

The Jira bridge does not use a scoped token; it authenticates as your user so
the required permissions are the ones of a normal Jira participant:

- **Jira Cloud**: an [API token][jira-cloud-api-token] generated from your
  Atlassian profile, sent together with your account email as username. The
  token inherits your account permissions.
- **Jira Data Center / Server**: an API token generated from your user profile
  (Data Center 8.0.0 and later, sent the same way as the Cloud token), or
  username and password ("session" authentication), which is the only option
  on older servers.

In all cases the account needs permission to view the project, and to create
and edit issues in it (view, comment and transition), typically through the
*Jira Software* project role or by being a participant in the project. Note
that closing/reopening issues is done by executing workflow *transitions*, so
the account must also be allowed to perform those transitions.

[jira-cloud-api-token]: https://id.atlassian.com/manage-profile/security/api-tokens

### Launchpad<a name="launchpad"></a>

No token or credentials are required. The Launchpad bridge is still
experimental (`launchpad-preview`) and read-only: it only queries the public
Launchpad API to import bugs and messages.

## Interacting with the bridge<a name="interacting-with-the-bridge"></a>

To push issues out to the bridge, run:

```bash
git bug bridge push [NAME]
```

To pull and integrate updates for issues from the bridge:

```bash
git bug bridge pull [NAME]
```

> [!TIP]
> See the [CLI documentation][doc/cli/bridge] for more information on the
> command line arguments and options.

The command line is primarily meant for programmatic usage or to interface with
`git-bug` with scripts or other tools. For day to day usage, we recommend taking
a look at [the supported interfaces][docs/usage/interfaces], which include a
robust TUI and an in-progress Web UI.

______________________________________________________________________

##### See more

- [A bird's-eye view of the internal architecture][docs/design/arch]
- [A description of the data model][docs/design/model]
- [An overview of the native interfaces][docs/usage/interfaces]
- [Filtering query results][docs/usage/filter]
- [Understanding the workflow models][docs/usage/workflows]
- :house: [Documentation home][docs/home]

[doc/cli/bridge]: ../md/git-bug_bridge.md
[docs/design/arch]: ../design/architecture.md
[docs/design/model]: ../design/data-model.md
[docs/feature-matrix]: ../feature-matrix.md
[docs/home]: ../README.md
[docs/usage/filter]: ./query-language.md
[docs/usage/interfaces]: ./interfaces.md
[docs/usage/workflows]: ./workflows.md
