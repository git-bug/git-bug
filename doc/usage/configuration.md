# Configuring `git-bug`<a name="configuring-git-bug"></a>

This page lists every configuration option `git-bug` understands, where each one
is read from, and what it defaults to.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Overview](#overview)
- [Configuration scopes](#configuration-scopes)
- [General options](#general-options)
- [Identity](#identity)
- [Bridges](#bridges)
- [Git options read by `git-bug`](#git-options-read-by-git-bug)

<!-- mdformat-toc end -->

## Overview<a name="overview"></a>

`git-bug` has no configuration file of its own. Everything lives in the standard
git configuration, under the `git-bug` section, and is read and written with the
regular `git config` command.

To see everything currently set for the repository you are in:

```bash
git config --get-regexp '^git-bug\.'
```

To set an option for the current repository, or for every repository:

```bash
git config git-bug.remote upstream
git config --global git-bug.webui.open false
```

Most `git-bug.*` keys are written for you by the command that owns them --
`git bug user adopt` writes `git-bug.identity`, `git bug bridge new` writes the
`git-bug.bridge.*` keys. The options in [General options](#general-options) are
the ones intended to be set by hand.

## Configuration scopes<a name="configuration-scopes"></a>

Git configuration is layered: the repository-local `.git/config`, your global
`~/.gitconfig`, and the system-wide file. `git-bug` does not read every key from
every layer.

| Keys                       | Read from          |
|----------------------------|--------------------|
| `git-bug.remote`           | local, then global |
| `git-bug.webui.open`       | local, then global |
| `git-bug.changes.notifier` | local, then global |
| `git-bug.identity`         | local only         |
| `git-bug.bridge.*`         | local only         |

Where both layers are consulted, the local value wins. Identity and bridge
configuration are deliberately local-only: they describe *this* repository, and
reading them from your global configuration would silently apply one
repository's identity or bridge to every other one.

`git bug wipe` removes the whole `git-bug` section from the local
configuration.

## General options<a name="general-options"></a>

| Key                  | Type   | Default  | Description                                                                                                                  |
|----------------------|--------|----------|------------------------------------------------------------------------------------------------------------------------------|
| `git-bug.remote`     | string | `origin` | The remote used by [`git bug pull`][doc/cli/pull] and [`git bug push`][doc/cli/push] when none is given on the command line. |
| `git-bug.webui.open` | bool   | `true`   | Whether [`git bug webui`][doc/cli/webui] opens the web UI in your default browser on startup.                                |

`git bug webui --no-open` suppresses the browser for a single run without
changing the configuration, and `--open` forces it on.

### Noticing changes made outside<a name="noticing-changes-made-outside"></a>

The commands that keep running, [`git bug webui`][doc/cli/webui] and
[`git bug termui`][doc/cli/termui], notice the changes made to the repository
outside of them, such as a `git push` into it or another `git-bug` command,
and show them shortly after. `git-bug.changes.notifier` chooses how:

| Value      | Behavior                                                                                                                                                                                                                                                                                                |
|------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `auto`     | The default: `watch` where it works, `poll` otherwise. If watching stops working while the command runs, it switches to `poll`.                                                                                                                                                                         |
| `watch`    | Be notified by the operating system as changes happen. Supported on Linux and Windows, on a local filesystem.                                                                                                                                                                                           |
| `poll`     | Check for changes every five seconds. Works everywhere, including on network filesystems (NFS, SMB), where watching sees nothing that another machine writes. On NFS, the client caches directory attributes, by default for 30 to 60 seconds, which delays changes made by another machine by as much. |
| `periodic` | Only rely on the full check made once a minute, see below.                                                                                                                                                                                                                                              |
| `none`     | Don't look for changes made outside at all: only those made through the command itself show up. For a repository that only `git-bug` writes to, through that one command.                                                                                                                               |

With any value but `none`, every change is also caught by a full check made
once a minute, so a wrong choice only delays the update, it never leaves
anything stale. Other commands don't need any of this: they check the
repository when they start.

## Identity<a name="identity"></a>

| Key                | Type      | Description                                        |
| ------------------ | --------- | -------------------------------------------------- |
| `git-bug.identity` | entity id | The identity used to author new bugs and comments. |

This key is managed by [`git bug user`][doc/cli/user]: `git bug user new` and
`git bug user adopt` write it, and `git bug user` shows the identity it points
to. It holds the id of an identity entity stored in the repository, so editing
it by hand is not useful -- use the commands instead.

Note that this is separate from `user.name` and `user.email`. Those are read
when *creating* an identity; `git-bug.identity` records which identity you then
went on to adopt.

## Bridges<a name="bridges"></a>

Each configured bridge stores its settings under `git-bug.bridge.<name>.`, where
`<name>` is the bridge name you chose when running
[`git bug bridge new`][doc/cli/bridge]. These keys are written by that command
and removed by `git bug bridge rm`; they are documented here so you can inspect
or adjust an existing bridge.

Every bridge has:

| Key                                    | Type      | Description                                                            |
|----------------------------------------|-----------|------------------------------------------------------------------------|
| `git-bug.bridge.<name>.target`         | string    | The bridge implementation: `github`, `gitlab`, `jira` or `launchpad`.  |
| `git-bug.bridge.<name>.lastImportTime` | timestamp | Bookkeeping for incremental imports. Written by `git bug bridge pull`. |

The remaining keys depend on the target.

#### GitHub<a name="github"></a>

| Key             | Description                                                  |
| --------------- | ------------------------------------------------------------ |
| `owner`         | The user or organisation owning the repository.              |
| `project`       | The repository name.                                         |
| `default-login` | The GitHub login used when no credential matches explicitly. |

#### GitLab<a name="gitlab"></a>

| Key             | Description                                                                 |
| --------------- | --------------------------------------------------------------------------- |
| `project-id`    | The numeric GitLab project id.                                              |
| `base-url`      | The GitLab instance URL, for self-hosted instances.                         |
| `default-login` | The GitLab login used when no credential matches explicitly.                |

#### Jira<a name="jira"></a>

| Key                      | Description                                                                           |
|--------------------------|---------------------------------------------------------------------------------------|
| `base-url`               | The Jira instance URL.                                                                |
| `project`                | The Jira project key.                                                                 |
| `default-login`          | The Jira login used when no credential matches explicitly.                            |
| `credentials-type`       | `SESSION` or `TOKEN`.                                                                 |
| `bug-id-map`             | Mapping from `git-bug` ids to Jira ids.                                               |
| `bug-id-revmap`          | The reverse mapping.                                                                  |
| `create-issue-defaults`  | Default fields for issues created by an export. Defaults to the Story type (`10001`). |
| `create-issue-gitbug-id` | If set, the Jira field to fill with the `git-bug` id when exporting.                  |

#### Launchpad<a name="launchpad"></a>

| Key       | Description                 |
|-----------|-----------------------------|
| `project` | The Launchpad project name. |

**Credentials are not stored in the git configuration.** Bridge tokens and
passwords live in your system keyring, managed by
[`git bug bridge auth`][doc/cli/bridge-auth]. A git repository is shared, and
its configuration is easy to leak; a token in `.git/config` would be a hazard.

## Git options read by `git-bug`<a name="git-options-read-by-git-bug"></a>

Beyond its own section, `git-bug` reads a handful of standard git options:

| Key                     | Used for                                                                             |
| ----------------------- | ------------------------------------------------------------------------------------ |
| `user.name`             | The name pre-filled when creating a new identity.                                    |
| `user.email`            | The email pre-filled when creating a new identity.                                   |
| `core.editor`           | The editor opened to compose messages.                                               |
| `remote.<name>.url`     | The remote to fetch `git-bug` refs from.                                             |
| `url.<base>.insteadOf`  | URL rewriting, applied to the remote URL.                                            |

For the editor, `git-bug` follows the same precedence as
[`git var`][git/var]: the `GIT_EDITOR` environment variable, then `core.editor`,
then `VISUAL`, then `EDITOR`, then the first of `editor`, `nano`, `vim`, `vi`
and `emacs` found on your `PATH`, and finally `ed`.

> [!NOTE]
> `git-bug` talks to remotes through the [go-git][go-git] library rather than by
> running the `git` binary, and go-git does not implement the whole of git's
> configuration. In particular, parts of your SSH setup (`~/.ssh/config`
> options such as `Port`, some `known_hosts` formats), http credential helpers,
> and `remote.<name>.pushurl` may not be honoured by `git bug pull` and
> `git bug push`. Because `git-bug` stores its data as ordinary git objects, you
> can always move them with plain `git` as a workaround -- see
> [discussion #1332][discuss/1332].

## Related reading<a name="related-reading"></a>

- [How to use bridges][docs/usage/bridges]
- [Filtering query results][docs/usage/filter]
- [Understanding the workflow models][docs/usage/workflows]
- :house: [Documentation home][docs/home]

[discuss/1332]: https://github.com/git-bug/git-bug/discussions/1332
[doc/cli/bridge]: ../md/git-bug_bridge_new.md
[doc/cli/bridge-auth]: ../md/git-bug_bridge_auth.md
[doc/cli/pull]: ../md/git-bug_pull.md
[doc/cli/push]: ../md/git-bug_push.md
[doc/cli/user]: ../md/git-bug_user.md
[doc/cli/termui]: ../md/git-bug_termui.md
[doc/cli/webui]: ../md/git-bug_webui.md
[docs/home]: ../README.md
[docs/usage/bridges]: ./third-party.md
[docs/usage/filter]: ./query-language.md
[docs/usage/workflows]: ./workflows.md
[git/var]: https://git-scm.com/docs/git-var
[go-git]: https://github.com/go-git/go-git
