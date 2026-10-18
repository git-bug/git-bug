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
  - [Verify a download](#verify-a-download)
- [Linux](#linux)
  - [Arch Linux](#arch-linux)
  - [Nixpkgs](#nixpkgs)
- [FreeBSD](#freebsd)
- [OpenBSD and NetBSD](#openbsd-and-netbsd)
- [MacOS](#macos)
- [Windows](#windows)
- [mise](#mise)
- [Build from source](#build-from-source)
- [Verify your installation](#verify-your-installation)

<!-- mdformat-toc end -->

## Download a pre-compiled release binary<a name="download-a-pre-compiled-release-binary"></a>

Each [release][rel/latest] provides, for every supported system:

- a **binary**: rename it to `git-bug` (`git-bug.exe` on Windows) and place it
  in a directory that's in your `PATH`.
- an **archive** (`tar.gz`, or `zip` on Windows): the same binary, plus man
  pages and shell completions.
- on Linux, **packages** that install all of the above where your system
  expects them.

The download links are in the section for your OS below. Binary links always
point to the latest release. Archive and package filenames include the version,
so their links point to v0.11.0; [the release page][rel/latest] may have a newer
one.

### Verify a download<a name="verify-a-download"></a>

Every release includes a `checksums.txt` file, which lists the SHA-256 of each
download. It is signed with keyless [cosign][cosign] by the release workflow,
and the signature is in `checksums.txt.sigstore.json`.

Download both files next to the file you want to check. The first command
below confirms that `checksums.txt` was signed by git-bug's release workflow.
The second checks your file against it:

```
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/git-bug/git-bug/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

sha256sum --ignore-missing -c checksums.txt
```

## Linux<a name="linux"></a>

`git-bug` is available on a variety of Linux distributions, but how you install
it depends on your distribution and package manager(s), as there is no standard
package manager common to all distributions.

You can download it from the release. The CPU is what `uname -m` prints:

| CPU       | Binary               | tar.gz               | deb                      | rpm                      | apk                      | Arch                      |
| --------- | :------------------: | :------------------: | :----------------------: | :----------------------: | :----------------------: | :-----------------------: |
| `x86_64`  | [⬇][bin/linux_amd64] | [⬇][arc/linux_amd64] | [⬇][pkg/linux_amd64.deb] | [⬇][pkg/linux_amd64.rpm] | [⬇][pkg/linux_amd64.apk] | [⬇][pkg/linux_amd64.arch] |
| `aarch64` | [⬇][bin/linux_arm64] | [⬇][arc/linux_arm64] | [⬇][pkg/linux_arm64.deb] | [⬇][pkg/linux_arm64.rpm] | [⬇][pkg/linux_arm64.apk] | [⬇][pkg/linux_arm64.arch] |
| `armv7l`  | [⬇][bin/linux_armv7] | [⬇][arc/linux_armv7] | [⬇][pkg/linux_armv7.deb] | [⬇][pkg/linux_armv7.rpm] | [⬇][pkg/linux_armv7.apk] | [⬇][pkg/linux_armv7.arch] |
| `armv6l`  | [⬇][bin/linux_armv6] | [⬇][arc/linux_armv6] | [⬇][pkg/linux_armv6.deb] | [⬇][pkg/linux_armv6.rpm] | [⬇][pkg/linux_armv6.apk] |                           |

Install a package with your package manager:

```
sudo apt install ./git-bug_<version>_linux_amd64.deb                # Debian, Ubuntu
sudo dnf install ./git-bug_<version>_linux_amd64.rpm                # Fedora, RHEL
sudo apk add --allow-untrusted ./git-bug_<version>_linux_amd64.apk  # Alpine
sudo pacman -U ./git-bug_<version>_linux_amd64.pkg.tar.zst          # Arch
```

### Arch Linux<a name="arch-linux"></a>

`git-bug` is available in the [official Arch Linux Package Repository][p/arch-repo]

<details><summary>Using <strong>pacman</strong></summary>

```sh
sudo pacman -S git-bug
```

</details>

`git-bug` is also available in the [Arch Linux User Repository (AUR)][p/aur], though may be outdated or orphaned.

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

Or download it from the release:

| CPU    | Binary                 | tar.gz                 |
| ------ | :--------------------: | :--------------------: |
| x86-64 | [⬇][bin/freebsd_amd64] | [⬇][arc/freebsd_amd64] |
| ARM64  | [⬇][bin/freebsd_arm64] | [⬇][arc/freebsd_arm64] |

## OpenBSD and NetBSD<a name="openbsd-and-netbsd"></a>

Download `git-bug` from the release:

| System         | Binary                 | tar.gz                 |
| -------------- | :--------------------: | :--------------------: |
| OpenBSD x86-64 | [⬇][bin/openbsd_amd64] | [⬇][arc/openbsd_amd64] |
| NetBSD x86-64  | [⬇][bin/netbsd_amd64]  | [⬇][arc/netbsd_amd64]  |

## MacOS<a name="macos"></a>

`git-bug` is shipped via [**Homebrew**][brew.sh]:

```
brew install git-bug
```

Or download it from the release:

| CPU           | Binary                | tar.gz                |
| ------------- | :-------------------: | :-------------------: |
| Apple Silicon | [⬇][bin/darwin_arm64] | [⬇][arc/darwin_arm64] |
| Intel         | [⬇][bin/darwin_amd64] | [⬇][arc/darwin_amd64] |

## Windows<a name="windows"></a>

`git-bug` is shipped via `scoop`:

```
scoop install git-bug
```

Or download it from the release:

| CPU    | Binary                 | zip                    |
| ------ | :--------------------: | :--------------------: |
| x86-64 | [⬇][bin/windows_amd64] | [⬇][arc/windows_amd64] |
| ARM64  | [⬇][bin/windows_arm64] | [⬇][arc/windows_arm64] |

## mise<a name="mise"></a>

On macOS, Windows, and Linux x86-64, `git-bug` can be installed with
[**mise**][mise]:

```
mise use -g aqua:git-bug/git-bug
```

## Build from source<a name="build-from-source"></a>

You'll need `git`, `go`, `make`, and optionally `node` and `pnpm` if you also build the web UI.
The versions we use are pinned in [`.tool-versions`](./.tool-versions).

```
git clone https://github.com/git-bug/git-bug.git
cd git-bug
git checkout v0.11.0 # optional: pick a release, or stay on the development head
make install
```

If you don't need the web UI, run `make install/noweb` instead: it doesn't need
`node` or `pnpm`, and `git bug webui` will report that it was not included.

`make install` places `git-bug` in your Go binary directory. Make sure it is in
your `PATH`, for example by adding this to `~/.zprofile` or `~/.bashrc`:

```
export PATH=$PATH:$(go env GOPATH)/bin
```

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

<!--
    When cutting a release, replace the previous version number everywhere in
    this file before tagging, so that the archive and package links point to the
    new release.
-->

[arc/darwin_amd64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_darwin_amd64.tar.gz
[arc/darwin_arm64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_darwin_arm64.tar.gz
[arc/freebsd_amd64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_freebsd_amd64.tar.gz
[arc/freebsd_arm64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_freebsd_arm64.tar.gz
[arc/linux_amd64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_amd64.tar.gz
[arc/linux_arm64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_arm64.tar.gz
[arc/linux_armv6]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv6.tar.gz
[arc/linux_armv7]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv7.tar.gz
[arc/netbsd_amd64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_netbsd_amd64.tar.gz
[arc/openbsd_amd64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_openbsd_amd64.tar.gz
[arc/windows_amd64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_windows_amd64.zip
[arc/windows_arm64]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_windows_arm64.zip
[bin/darwin_amd64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_darwin_amd64
[bin/darwin_arm64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_darwin_arm64
[bin/freebsd_amd64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_freebsd_amd64
[bin/freebsd_arm64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_freebsd_arm64
[bin/linux_amd64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_linux_amd64
[bin/linux_arm64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_linux_arm64
[bin/linux_armv6]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_linux_armv6
[bin/linux_armv7]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_linux_armv7
[bin/netbsd_amd64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_netbsd_amd64
[bin/openbsd_amd64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_openbsd_amd64
[bin/windows_amd64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_windows_amd64.exe
[bin/windows_arm64]: https://github.com/git-bug/git-bug/releases/latest/download/git-bug_windows_arm64.exe
[brew.sh]: https://brew.sh
[cosign]: https://docs.sigstore.dev/cosign/system_config/installation/
[docs/home]: ./doc
[mise]: https://mise.jdx.dev
[p/arch-repo]: https://archlinux.org/packages/extra/x86_64/git-bug/
[p/aur]: https://aur.archlinux.org/packages/git-bug-bin
[p/nix]: https://github.com/NixOS/nixpkgs/blob/nixos-unstable/pkgs/applications/version-management/git-bug/default.nix
[pkg/linux_amd64.apk]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_amd64.apk
[pkg/linux_amd64.arch]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_amd64.pkg.tar.zst
[pkg/linux_amd64.deb]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_amd64.deb
[pkg/linux_amd64.rpm]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_amd64.rpm
[pkg/linux_arm64.apk]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_arm64.apk
[pkg/linux_arm64.arch]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_arm64.pkg.tar.zst
[pkg/linux_arm64.deb]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_arm64.deb
[pkg/linux_arm64.rpm]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_arm64.rpm
[pkg/linux_armv6.apk]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv6.apk
[pkg/linux_armv6.deb]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv6.deb
[pkg/linux_armv6.rpm]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv6.rpm
[pkg/linux_armv7.apk]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv7.apk
[pkg/linux_armv7.arch]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv7.pkg.tar.zst
[pkg/linux_armv7.deb]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv7.deb
[pkg/linux_armv7.rpm]: https://github.com/git-bug/git-bug/releases/download/v0.11.0/git-bug_0.11.0_linux_armv7.rpm
[rel/latest]: https://github.com/git-bug/git-bug/releases/latest
