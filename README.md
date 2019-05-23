<p align="center">
    <img width="150px" src="https://cdn.rawgit.com/MichaelMure/git-bug/master/misc/logo/logo-alpha-flat-bg.svg">
</p>
<h1 align="center">git-bug</h1>

<div align="center">

[![Build Status](https://travis-ci.org/MichaelMure/git-bug.svg?branch=master)](https://travis-ci.org/MichaelMure/git-bug)
[![Backers on Open Collective](https://opencollective.com/git-bug/backers/badge.svg)](#backers) [![Sponsors on Open Collective](https://opencollective.com/git-bug/sponsors/badge.svg)](#sponsors) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3+-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)
[![GoDoc](https://godoc.org/github.com/MichaelMure/git-bug?status.svg)](https://godoc.org/github.com/MichaelMure/git-bug)
[![Go Report Card](https://goreportcard.com/badge/github.com/MichaelMure/git-bug)](https://goreportcard.com/report/github.com/MichaelMure/git-bug)
[![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/the-git-bug/Lobby)

</div>

`git-bug` is a bug tracker that:
- **fully embed in git**: you only need your git repository to have a bug tracker
- **is distributed**: use your normal git remote to collaborate, push and pull your bugs !
- **works offline**: in a plane or under the sea ? keep reading and writing bugs
- **prevent vendor locking**: your usual service is down or went bad ? you already have a full backup
- **is fast**: listing bugs or opening them is a matter of milliseconds
- **doesn't pollute your project**: no files are added in your project
- **integrate with your tooling**: use the UI you like (CLI, terminal, web) or integrate with your existing tools through the CLI or the GraphQL API
- **bridge with other bug trackers**: [bridges](#bridges) exist to import and soon export to other trackers.

:construction: This is now more than a proof of concept, but still not fully stable. Expect dragons and unfinished business. :construction:

## Install

<details><summary>Pre-compiled binaries</summary>

1. Go to the [release page](https://github.com/MichaelMure/git-bug/releases/latest) and download the appropriate binary for your system.
2. Copy the binary anywhere in your PATH
3. Rename the binary to `git-bug` (or `git-bug.exe` on windows)

That's all !

</details>

<details><summary>Linux packages</summary>

* [Archlinux (AUR)](https://aur.archlinux.org/packages/?K=git-bug)

</details>

<details><summary>go get (unstable)</summary>

```shell
go get -u github.com/MichaelMure/git-bug
```

If it's not done already, add golang binary directory in your PATH:

```bash
export PATH=$PATH:$(go env GOROOT)/bin:$(go env GOPATH)/bin
```

</details>

## CLI usage

Create a new identity:

```
git bug user create
```

Create a new bug:

```
git bug add
```

Your favorite editor will open to write a title and a message.

You can push your new entry to a remote:
```
git bug push [<remote>]
```

And pull for updates:
```
git bug pull [<remote>]
```

List existing bugs:
```
git bug ls
```

Filter and sort bugs using a [query](doc/queries.md):
```
git bug ls "status:open sort:edit"
```

You can now use commands like `show`, `comment`, `open` or `close` to display and modify bugs. For more details about each command, you can run `git bug <command> --help` or read the [command's documentation](doc/md/git-bug.md).

## Interactive terminal UI

An interactive terminal UI is available using the command `git bug termui` to browse and edit bugs.

![Termui recording](misc/termui_recording.gif)

## Web UI (status: WIP)

You can launch a rich Web UI with `git bug webui`.

![Web UI screenshot 1](misc/webui1.png)
![Web UI screenshot 2](misc/webui2.png)

This web UI is entirely packed inside the same go binary and serve static content through a localhost http server.

The web UI interact with the backend through a GraphQL API. The schema is available [here](graphql/).

## Bridges

### Importer implementations

|                                                 | Github             | Launchpad          |
| ----------------------------------------------- | :----------------: | :----------------: |
| **incremental**<br/>(can import more than once) | :heavy_check_mark: | :x:                |
| **with resume**<br/>(download only new data)    | :x:                | :x:                |
| **identities**                                  | :heavy_check_mark: | :heavy_check_mark: |
| identities update                               | :x:                | :x:                |
| **bug**                                         | :heavy_check_mark: | :heavy_check_mark: |
| comments                                        | :heavy_check_mark: | :heavy_check_mark: |
| comment editions                                | :heavy_check_mark: | :x:                |
| labels                                          | :heavy_check_mark: | :x:                |
| status                                          | :heavy_check_mark: | :x:                |
| title edition                                   | :heavy_check_mark: | :x:                |
| **automated test suite**                        | :x:                | :x:                |

### Exporter implementations

Todo !

## Internals

Interested by how it works ? Have a look at the [data model](doc/model.md) and the [internal bird-view](doc/architecture.md).

## Misc

- [Bash completion](misc/bash_completion)
- [Zsh completion](misc/zsh_completion)
- [ManPages](doc/man)

## Planned features

- media embedding
- exporter to github issue
- extendable data model to support arbitrary bug tracker
- inflatable raptor

## Contribute

PRs accepted. Drop by the [Gitter lobby](https://gitter.im/the-git-bug/Lobby) for a chat or browse the issues to see what is worked on or discussed.

Developers unfamiliar with Go may try to clone the repository using "git clone". Instead, one should use:

```shell
go get -u github.com/MichaelMure/git-bug
```

The git repository will then be available:

```shell
# Note that $GOPATH defaults to $HOME/go
$ cd $GOPATH/src/github.com/MichaelMure/git-bug/
```

You can now run `make` to build the project, or `make install` to install the binary in `$GOPATH/bin/`.

To work on the web UI, have a look at [the dedicated Readme.](webui/Readme.md)


## Contributors :heart:

This project exists thanks to all the people who contribute.
<a href="https://github.com/MichaelMure/git-bug/graphs/contributors"><img src="https://opencollective.com/git-bug/contributors.svg?width=890&button=false" /></a>


## Backers

Thank you to all our backers! 🙏 [[Become a backer](https://opencollective.com/git-bug#backer)]

<a href="https://opencollective.com/git-bug#backers" target="_blank"><img src="https://opencollective.com/git-bug/tiers/backer.svg?width=890"></a>


## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website. [[Become a sponsor](https://opencollective.com/git-bug#sponsor)]

<a href="https://opencollective.com/git-bug/sponsor/0/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/1/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/2/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/3/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/4/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/5/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/6/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/7/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/8/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/9/website" target="_blank"><img src="https://opencollective.com/git-bug/tiers/sponsor/9/avatar.svg"></a>


## License

Unless otherwise stated, this project is released under the [GPLv3](LICENSE) or later license © Michael Muré.

The git-bug logo by [Viktor Teplov](https://github.com/vandesign) is released under the [Creative Commons Attribution 4.0 International (CC BY 4.0)](misc/logo/LICENSE) license © Viktor Teplov.
