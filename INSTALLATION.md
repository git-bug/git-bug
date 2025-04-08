<!--
    NOTE TO CONTRIBUTORS:

    We use HTML elements within <details> in order to avoid parsing errors with
    GFM caused by triple-backtick blocks or alert elements being nested next to
    the summary or beginning of the <details> block.

    Please keep this in mind as you make changes.
-->

# Installation Guide<a name="installation-guide"></a>

`git-bug` is distributed as a single binary, and is available for multiple
platforms. Follow this document for instructions on how to install `git-bug`,
and verify your installation.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Download a pre-compiled release binary](#download-a-pre-compiled-release-binary)
- [Linux](#linux)
  - [Arch Linux](#arch-linux)
  - [Nixpkgs](#nixpkgs)
- [FreeBSD](#freebsd)
- [MacOS](#macos)
- [Windows](#windows)
- [Build from source](#build-from-source)
- [Verify your installation](#verify-your-installation)

<!-- mdformat-toc end -->

## Download a pre-compiled release binary<a name="download-a-pre-compiled-release-binary"></a>

You can download the latest release binary from [the release page][rel/latest],
making sure to grab the appropriate binary for your system.

Next, rename the binary to `git-bug`, or `git-bug.exe` if you're using Windows.

Finally, place the binary in a directory that's in your `PATH`. That's it! You
should now have `git-bug` available as a command line tool.

## Linux<a name="linux"></a>

`git-bug` is available on a variety of Linux distributions, but how you install
it depends on your distribution and package manager(s), as there is no standard
package manager common to all distributions.

### Arch Linux<a name="arch-linux"></a>

`git-bug` is available in the [Arch Linux User Repository (AUR)][p/aur].

Below, you'll find a **non-exhaustive** list of commands that use common third
party tools for installing packages from the AUR.

<details><summary>Using <strong>aurutils</strong></summary>
<pre>aur sync git-bug-bin && pacman -Syu git-bug-bin</pre>
</details>

<details><summary>Using <strong>yay</strong></summary>
<pre>yay -S git-bug-bin</pre>
</details>

### Nixpkgs<a name="nixpkgs"></a>

`git-bug` is available via [nixpkgs][p/nix].

<details><summary>Using <strong>home-manager</strong></summary>
<pre>
home.package = with pkgs; [
  git-bug
];
</pre>
</details>

<details><summary>Using <strong>system configuration</strong></summary>
<pre>
environment.systemPackages = with pkgs; [
  git-bug
];
</pre>
</details>

<details><summary>Using <strong>nix profile</strong></summary>
<pre>nix profile install nixpkgs\#git-bug</pre>
</details>

<details><summary>Temporary installation with <strong>nix shell</strong></summary>
<pre>
nix shell nixpkgs\#git-bug
</pre>
</details>

## FreeBSD<a name="freebsd"></a>

`git-bug` is available on FreeBSD through a few different methods.

<details><summary>Using <strong>pkg</strong></summary>
<pre>pkg install git-bug</pre>
</details>

<details><summary>Using the <strong>ports</strong> collection</summary>
<pre>make -C /usr/ports/devel/git-bug install clean</pre>
</details>

## MacOS<a name="macos"></a>

`git-bug` is shipped via [**Homebrew**][brew.sh]:

```
brew install git-bug
```

## Windows<a name="windows"></a>

`git-bug` is shipped via `scoop`:

```
scoop install git-bug
```

## Build from source<a name="build-from-source"></a>

You can also build `git-bug` from source, if you wish. You'll need the following
dependencies:

- `git`
- `go`
- `make`

Ensure that the `go` binary directory (`$GOPATH/bin`) is in your `PATH`. It is
recommended to set this within your shell configuration file(s), such as
`~/.zprofile` or `~/.bashrc`.

```
export PATH=$PATH:$(go env GOROOT)/bin:$(go env GOPATH)/bin
```

> [!NOTE]
> The commands below assume you do not want to keep the repository on disk, and
> thus clones the repository to a new temporary directory and performs a
> lightweight clone in order to reduce network latency and data transfer.
>
> As a result, the repository cloned during these steps will not contain the
> full history. If that is important to you, clone the repository using the
> method you prefer, check out your preferred revision, and run `make install`.

**First, create a new repository on disk:**

```
cd $(mktemp -d) && git init .
```

**Next, set the remote to the upstream source:**

```
git remote add origin git@github.com:git-bug/git-bug.git
```

Next, choose whether you want to build from a release tag, branch, or
development head and expand the instructions below.

<details><summary>Build from <strong>a release tag</strong></summary>

First, list all of the tags from the repository (we use `sed` in the command
below to filter out some unnecessary visual noise):

<pre>
git ls-remote origin refs/tags/\* | sed -e 's/refs\/tags\///'
</pre>

You'll see output similar to:

<pre>
c1a08111b603403d4ee0a78c1214f322fecaa3ca        0.1.0
d959acc29dcbc467790ae87389f9569bb830c8c6        0.2.0
ad59f77fd425b00ae4b8d7360a64dc3dc1c73bd0        0.3.0
...
</pre>

<blockquote><strong>Tip</strong><p>
The <em>tags</em> are in the right-most column. Old revisions up to and
including <code>0.7.1</code> do not contain a <em>v</em> prefix, however, all
revisions after, do.
</p></blockquote>

Select the tag you wish to build, and fetch it using the command below. Be sure
to replace <code>REPLACE-ME</code> with the tag you selected:

<pre>
git fetch --no-tags --depth 1 origin +refs/tags/REPLACE-ME:refs/tags/REPLACE-ME
</pre>

<blockquote><strong>NOTE</strong><p>
The <code>--no-tags</code> flag might seem out of place, since we <em>are</em>
fetching a tag, but it isn't -- the reason we use this is avoid fetching other
tags, in case you have <code>fetch.pruneTags</code> enabled in your global
configuration, which causes <code>git</code> to fetch <em>all</em> tags.
</p></blockquote>

Next, check out the tag, replacing <code>REPLACE-ME</code> with the tag you
selected:

<pre>
git checkout REPLACE-ME
</pre>

Finally, run the <code>install</code> target from <code>//:Makefile</code>:

<pre>
make install
</pre>

This will build <code>git-bug</code> and place it in your Go binary directory.

</details>

<details>
<summary>
Build the <strong>unstable development <code>HEAD</code></strong>
</summary>

First, fetch the most recent commit for the default branch:

<pre>
git fetch --no-tags --depth 1 origin HEAD:refs/remotes/origin/HEAD
</pre>

Next, check out the tree you pulled:

<pre>
git checkout origin/HEAD
</pre>

Finally, run the <code>install</code> target from <code>//:Makefile</code>:

<pre>
make install
</pre>

This will build <code>git-bug</code> and place it in your Go binary directory.

</details>

## Verify your installation<a name="verify-your-installation"></a>

To verify that `git-bug` was installed correctly, you can run the following
command. If you see output similar to what's shown below (and without any
errors), you're all set!

```
git bug version
```

______________________________________________________________________

##### See more

- [Documentation home][docs/home]

[brew.sh]: https://brew.sh
[docs/home]: ./doc
[p/aur]: https://aur.archlinux.org/packages/git-bug-bin
[p/nix]: https://github.com/NixOS/nixpkgs/blob/nixos-unstable/pkgs/applications/version-management/git-bug/default.nix
[rel/latest]: https://github.com/git-bug/git-bug/releases/latest
