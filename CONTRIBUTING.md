# Contributing

:wave: Hey there! Thanks for considering taking the time to contribute to
`git-bug`. This page contains some general guidelines, and instructions for
getting started as a contributor to this project.

## Get the source code

Clone this repository to your system in a way you're comfortable with. Below, we
show a command that [clones the repository][how-to-clone] using SSH, and places
it in `~/code/git-bug`.

```
git clone git@github.com:git-bug/git-bug ~/code/git-bug
```

> **Important**
>
> If you wish to clone the repository to another location on disk, change
> `~/code/git-bug` to your desired path. The rest of this documentation will
> refer to `~/code/git-bug` in all instances, so make sure you change them
> there, too.

## Software recommendations

While you can install Golang and hack on this project on your own, you're likely
to have a better experience if you install the following software.

### nix (_recommended_)

[`nix`][install/nix] is used in this repository to provide a common development
shell, with a complete set of the appropriate version of the tools used to work
on this project.

To install `nix`, you can follow [the official instructions][install/nix], but we
recommend adding some additional flags:

```
curl -L https://nixos.org/nix/install | sh -s -- --daemon \
  --nix-extra-conf-file <( echo "experimental-features = nix-command flakes")
```

### direnv (_recommended_)

[`direnv`][install/direnv] is used to automatically activate the development
shell (because of the `.envrc` in the root of this repository). It can be
installed either with nix, or independently.

<details>
<summary><strong>With nix</strong></summary>

```
nix --extra-experimental-options 'flakes nix-command' profile install nixpkgs\#direnv
```

</details>

<details>
<summary><strong>Without nix</strong></summary>

You can install `direnv` by following [the official
instructions][install/direnv]. There's a second step that is critical -- be sure
to [configure your shell][install/direnv/shell].

</details>

There's a second step that is critical -- be sure to [configure your
shell][install/direnv/shell]. After installation, you'll need to start a new
shell.

#### <a name="direnv-configuration"></a> Configuring direnv

If you install `direnv`, it is recommended to set the following configuration
options to improve your user experience. At the time of writing, these go in
`~/.config/direnv/direnv.toml`.

This configuration, namely, the `whitelist.exact` property, will ensure that
`direnv` always automatically sources the `.envrc` in this repository. The other
configuration properties are recommended, but only to reduce visual noise.

```
hide_env_diff = true
warn_timeout = 0

[whitelist]
exact = ["~/code/git-bug/.envrc"]
```

[install/nix]: https://nix.dev/install-nix
[install/direnv]: https://github.com/direnv/direnv/blob/master/docs/installation.md
[install/direnv/shell]: https://github.com/direnv/direnv/blob/master/docs/hook.md
