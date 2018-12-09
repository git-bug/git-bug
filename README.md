<p align="center">
    <img width="150px" src="https://cdn.rawgit.com/MichaelMure/git-bug/master/misc/logo/logo-alpha-flat-bg.svg">
</p>
<h1 align="center">git-bug</h1>

<div align="center">

Distributed bug tracker embedded in Git.

[![Build Status](https://travis-ci.org/MichaelMure/git-bug.svg?branch=master)](https://travis-ci.org/MichaelMure/git-bug)
[![Backers on Open Collective](https://opencollective.com/git-bug/backers/badge.svg)](#backers) [![Sponsors on Open Collective](https://opencollective.com/git-bug/sponsors/badge.svg)](#sponsors) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3+-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)
[![GoDoc](https://godoc.org/github.com/MichaelMure/git-bug?status.svg)](https://godoc.org/github.com/MichaelMure/git-bug)
[![Go Report Card](https://goreportcard.com/badge/github.com/MichaelMure/git-bug)](https://goreportcard.com/report/github.com/MichaelMure/git-bug)
[![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/the-git-bug/Lobby)

</div>

Would it be nice to not have to rely on a web service somewhere to deal with bugs ?

Would it be nice to be able to browse and edit bug reports offline?

`git-bug` is a distributed bug tracker embedded in `git`. It uses git's internal storage so **no files are added in your project**.

As you would do with commits and branches, you can push your bugs to the same git remote you are already using to collaborate with other people.

:construction: This is for now a proof of concept. Expect dragons and unfinished business. :construction:

## Install

### Go get
```shell
go get -u github.com/MichaelMure/git-bug
```

If it's not done already, add golang binary directory in your PATH:

```bash
export PATH=$PATH:$(go env GOROOT)/bin:$(go env GOPATH)/bin
```

### Pre-compiled binaries

1. Go to the [release page](https://github.com/MichaelMure/git-bug/releases/latest) and download the appropriate binary for your system.
2. Copy the binary anywhere in your PATH
3. Rename the binary to `git-bug` (or `git-bug.exe` on windows)

That's all !

### Linux packages

* [Archlinux (AUR)](https://aur.archlinux.org/packages/?K=git-bug)

## CLI usage

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

![Termui recording](doc/termui_recording.gif)

## Web UI (status: WIP)

You can launch a rich Web UI with `git bug webui`.

![Web UI screenshot 1](doc/webui1.png)
![Web UI screenshot 2](doc/webui2.png)

This web UI is entirely packed inside the same go binary and serve static content through a localhost http server.

The web UI interact with the backend through a GraphQL API. The schema is available [here](graphql/schema.graphql).

## Internals

Interested by how it works ? Have a look at the [data model](doc/model.md).

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


## Contributors

This project exists thanks to all the people who contribute. 
<a href="https://github.com/MichaelMure/git-bug/graphs/contributors"><img src="https://opencollective.com/git-bug/contributors.svg?width=890&button=false" /></a>


## Backers

Thank you to all our backers! 🙏 [[Become a backer](https://opencollective.com/git-bug#backer)]

<a href="https://opencollective.com/git-bug#backers" target="_blank"><img src="https://opencollective.com/git-bug/backers.svg?width=890"></a>


## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website. [[Become a sponsor](https://opencollective.com/git-bug#sponsor)]

<a href="https://opencollective.com/git-bug/sponsor/0/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/1/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/2/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/3/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/4/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/5/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/6/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/7/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/8/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/git-bug/sponsor/9/website" target="_blank"><img src="https://opencollective.com/git-bug/sponsor/9/avatar.svg"></a>



## License

Unless otherwise stated, this project is released under the [GPLv3](LICENSE) or later license © Michael Muré.

The git-bug logo by [Viktor Teplov](https://github.com/vandesign) is released under the [Creative Commons Attribution 4.0 International (CC BY 4.0)](misc/logo/LICENSE) license © Viktor Teplov.
