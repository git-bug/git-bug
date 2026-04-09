# Contributing <a name="contributing"></a>

:wave: Hey there! Thanks for considering taking the time to contribute to
`git-bug`. This page contains some general guidelines, and instructions for
getting started as a contributor to this project.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Get the source code](#get-the-source-code)
- [Software requirements](#software-requirements)
  - [Install nix](#install-nix)
  - [Install and configure `direnv`](#install-and-configure-direnv)
- [3.0 | Post-installation tasks](#30--post-installation-tasks)
  - [3.1 | Open a new shell](#31--open-a-new-shell)
  - [3.2 | Test the development shell](#32--test-the-development-shell)
- [Useful development commands](#useful-development-commands)
- [Submitting changes](#submitting-changes)
  - [Commit messages are the source of truth](#commit-messages-are-the-source-of-truth)
  - [Push early, push often](#push-early-push-often)
  - [Pull requests are squashed](#pull-requests-are-squashed)
  - [Draft vs Ready](#draft-vs-ready)
  - [Singular unit of change](#singular-unit-of-change)

<!-- mdformat-toc end -->

## Get the source code<a name="get-the-source-code"></a>

Clone this repository to your system in a way you're comfortable with. Below, we
show a command that clones the repository using SSH, and places it in
`~/code/git-bug`.

```bash
git clone git@github.com:git-bug/git-bug ~/code/git-bug
```

> [!TIP]
> If you wish to clone the repository to another location on disk, change
> `~/code/git-bug` to your desired path (e.g. your current directory can be used
> with `.` or by omitting the path argument). The rest of this documentation
> will refer to `~/code/git-bug` in all instances, so make sure you change them
> there to match the location you've cloned the repository.

## Software requirements<a name="software-requirements"></a>

This repository uses `nix` to provide a consistent development environment,
ensuring that each contributor has the same revision of each dependency and tool
installed. It is **strongly** encouraged to use `nix` when contributing to
`git-bug` to remove the "it works on my machine" class of errors, and ensure you
have a smoother experience passing the CI pipelines (wrt formatting and such).

While you can manually install the [tools and packages we use](./shell.nix) and
hack on this project on your own, you will miss out on the hermeticity and
standardization that our development shell provides. You may end up fighting
with failing CI pipelines more often, or have to figure out how to perform
various tasks on your own. Using the development shell ensures you always have
every tool you need to contribute to `git-bug`, and that each tool is always
configured correctly.

Because of this, we encourage all contributors to follow the documentation below
to install the dependencies for the development shell.

> [!NOTE]
> In the future, we will provide a container image with `nix` pre-installed and
> everything configured to get you started. This will be able to be pulled like
> any other image, and will be made compatible with VSCode's "devcontainer"
> feature and GitHub Codespaces.
>
> For more information, see the [tracking issue][issue/1364].

______________________________________________________________________

### Install nix<a name="install-nix"></a>

To install `nix`, you can follow [the official instructions][install/nix].

We recommend following the instructions for `multi-user mode` where applicable.

> [!IMPORTANT]
> The rest of this document assumes you have successfully installed `nix`.

______________________________________________________________________

### Install and configure `direnv`<a name="install-and-configure-direnv"></a>

[`direnv`][install/direnv] can be used to automatically activate the development
shell (using [`//:.envrc`][envrc]). It can be installed either with `nix`, or
independently.

<details>
<summary><strong>With nix-env</strong> <em>(suggested for users new to nix)</em></summary>

```bash
nix-env -iA direnv
```

Next, run the following commands to apply the **optional** configuration for
direnv. Be sure to change references to `~/code/git-bug` if you have cloned the
repository somewhere else.

> [!NOTE]
> The above command installs direnv in a non-deterministic way on your host
> system. To update it, you'll need to run `nix-env -uA direnv`. See
> `man nix-env-upgrade` for more information, or choose to install `direnv` in
> another way that is more compatible with the approach you take for your
> system.

<strong>Create a configuration file for <code>direnv</code></strong>

```bash
mkdir -p ~/.config/direnv && touch ~/.config/direnv/direnv.toml
```

<strong>Disable the warning for shells with longer load times</strong>

_This is optional, but recommended, as it helps reduce visual clutter._

```bash
nix-shell -p dasel --run \
	'dasel -- -r toml -f ~/.config/direnv/direnv.toml put -t int -v 0 ".global.warn_timeout"'
```

<strong>Disable printing of the environment variables that change</strong>

_This is optional, but recommended, as it helps reduce visual clutter._

```bash
nix-shell -p dasel --run \
	'dasel -- -r toml -f ~/.config/direnv/direnv.toml put -t bool -v true ".global.hide_env_diff"'
```

<strong>Configure automatic activation of the development shell</strong>

_This is optional, but strongly recommended._

```bash
nix-shell -p dasel --run \
	'dasel -- -r toml -f ~/.config/direnv.toml put -v "~/code/git-bug/.envrc" ".whitelist.exact[]"'
```

Alternatively, simply run `direnv allow` after moving into the repository for
the first time.

> **IMPORTANT**<br /> If you choose not to allow the shell to be automatically
> activated, you will need to type `nix-shell` every time you want to activate
> it, and this will swap you into bash and change your prompt. You'll have a far
> better experience allowing `direnv` to automatically manage activation and
> deactivation.

<strong>Configure your shell</strong>

This final step is crucial -- be sure to
[configure your shell][install/direnv/shell] for direnv, so that it is loaded
properly on shell initialization.

</details>

<details>
<summary><strong>Using <code>home-manager</code></strong></summary>

```nix
{
  programs.direnv = {
    enable = true;
    nix-direnv.enable = true;

    # one of the following, depending on your shell
    # enableZshIntegration = true;
    # enableBashIntegration = true;
    # enableFishIntegration = true;
    # enableNushellIntegration = true;

    config = {
      hide_env_diff = true;
      warn_timeout = 0;

      whitelist.exact = [ "~/code/git-bug/.envrc" ];
    };
  };
}
```

</details>

______________________________________________________________________

## 3.0 | Post-installation tasks<a name="30--post-installation-tasks"></a>

Congratulations! If you've reached this section of the documentation, chances
are that you have a working development environment for contributing to this
repository. Read below for some additional tasks you should complete.

### 3.1 | Open a new shell<a name="31--open-a-new-shell"></a>

In order for the installation to take effect, you will need to open a new shell.
It is recommended to do this and complete the test (described below) prior to
closing the shell you ran the installation script in, just in case you run into
issues and need to refer to any output it provided.

### 3.2 | Test the development shell<a name="32--test-the-development-shell"></a>

To test that the development shell is active, you will need to move to the
repository's directory. If you installed and properly configured `direnv` for
automatic activation, the shell should activate upon changing directories.

```bash
{ test -n "$IN_NIX_SHELL" && echo "ACTIVE"; } || echo "INACTIVE"
```

If you have activated the development shell, you will see `ACTIVE` printed to
the console. If you have not, you will see `INACTIVE` printed to the console. If
you did configure direnv to automatically load the shell, you may need to type
`direnv allow`. If you chose to not use `direnv`, you will need to manually type
`nix-shell`.

______________________________________________________________________

## Useful development commands<a name="useful-development-commands"></a>

- `make build` - build `git-bug` and output the binary at `./result/bin/git-bug`
- `make build/debug` - build a debugger-friendly binary build\` for rapid
  iteration,
- `nix-build -A default` - build `git-bug` for your platform and output the
  binary at `./result/bin/git-bug`. You can optionally tag it with a version by
  adding
  `--argstr version $(git describe --match 'v*' --always --dirty --broken)`.
  - add `--arg debug true` to skip the test suite and build a debugger-friendly
    binary that fetches the web interface from the filesystem, and contains the
    symbol table and `DWARF` debug info
  - optionally tag it with a version by adding the following flag:
    `--argstr version $(git describe --match 'v*' --always --dirty --broken)`
- `nix-shell --run treefmt` - format everything
  - `nix-shell --run 'treefmt <path...>'` to restrict the scope to given
    directories or files
  - _see `nix-shell --run 'treefmt --help'` for more information_
- `make check` to run tests and lint/format checks
- `make format` to format all files
- `go generate` - generate cli documentation and shell completion files
  - this is automatically executed by many `make` targets, e.g. `make build`
- `go test ./commands -update` - update golden files used in tests
  - this is _required_ when changing the output of CLI commands, if the files in
    `//commands/testdata/...` do not match the new output format
- `pinact` to pin any newly-added github action libraries
  - `pinact upgrade` to upgrade action libraries

> [!NOTE]
> There is an ongoing effort to simplify the commands you need to call in our
> environment, with a trend toward `nix`, while `make` may continue to be
> supported for common workflows (e.g. building a release binary).

## Submitting changes<a name="submitting-changes"></a>

You can submit your changes in the typical fork-based workflow to this
repository on GitHub. That is: fork this repository, push to a branch to your
repository, and create a pull request.

If you are in the development shell, you have the `gh` command line tool
available for use with github.

### Commit messages are the source of truth<a name="commit-messages-are-the-source-of-truth"></a>

Commit subjects and messages are the source of truth for a variety of things,
including the public-facing changelog ([`//:CHANGELOG.md`][changelog]) and
release descriptions. Writing thorough commit messages is beneficial to anyone
reviewing a commit, including future you!

Please be sure to read the [commit message guidelines][doc/contrib/commit].

### Push early, push often<a name="push-early-push-often"></a>

We encourage pushing small changes. There is no such thing as a contribution
being "too small". If you're working on a big feature or large refactor, we
encourage you to think about how to introduce it incrementally, in byte-sized
chunks, and submit those smaller changes frequently.

### Pull requests are _squashed_<a name="pull-requests-are-squashed"></a>

All Pull Requests in this repository are _squash merged_. The resulting commit's
subject and body is set as such:

- The subject is set to the PR title, appended with the PR's number
- The body is set to the contents of the PR description, otherwise known as the
  first comment in the "Discussion" tab

As such, maintainers may edit your PR title and description (the initial
comment) before merging. This is to make sure commits that land in `trunk`
messages have the appropriate context.

### Draft vs Ready<a name="draft-vs-ready"></a>

Pull Requests can be marked as "ready for review", or as a "draft". Drafts are
less likely to get reviewed, for the simple fact that in English, the word
"draft" (in relation to a change) implies that the work is not yet ready, or
still being worked on. That said, we encourage you to iterate locally,
especially if you are in the earlier stages of a change.

Any PR, even a draft, triggers the pipelines to run our test suite. Note,
however, that due to the cost (in terms of time and finance) associated with
running the pipelines, we may, at our discretion, exclude various parts of our
test suite from draft PRs.

### Singular unit of change<a name="singular-unit-of-change"></a>

Please keep any one PR limited to a singular unit of change. That is, do your
best to avoid making multiple different changes in the same tree / PR. This
helps make it simpler to reason about a change during review, and reduces the
chance that your proposal ends up spawning five different conversations that
each dig down their own rabbit hole.

Good examples of a singular unit of change:

- Renaming a function (or method or class or interface or...)
- Adding a new endpoint
- Deprecating a thing
- Removing all of one type of thing (e.g. "remove all unused functions")

Aim to make sure that your change does _one_ thing.

Examples of what we _do not_ want to see in a given PR:

- Rename function Foo and add function Bar
- Update Baz and Foo
- Move Thing and upgrade Bar
- Deprecate Zam and add a new function Zoo

Generally speaking, look out for the word "and". If you find yourself needing to
write this word in a commit message when describing what your change does,
consider whether it would be best to split those changes up into individual
trees.

______________________________________________________________________

##### See more

- [View the commit message guidelines][doc/contrib/commit]
- [An overview of the architecture][doc/design/arch]
- [Learn about the data model][doc/design/model]
- [See how to create a new entity][example-entity]

[changelog]: ./CHANGELOG.md
[doc/contrib/commit]: ./doc/contrib/commit-message-guidelines.md
[doc/design/arch]: ./doc/design/architecture.md
[doc/design/model]: ./doc/design/data-model.md
[envrc]: ./.envrc
[example-entity]: ./entity/dag/example_test.go
[install/direnv]: https://github.com/direnv/direnv/blob/trunk/docs/installation.md
[install/direnv/shell]: https://github.com/direnv/direnv/blob/trunk/docs/hook.md
[install/nix]: https://nix.dev/install-nix
[issue/1364]: https://github.com/git-bug/git-bug/issues/1364
