<div align="center">

<img width="64px" src="https://cdn.rawgit.com/git-bug/git-bug/trunk/misc/logo/logo-alpha-flat-bg.svg">

# git-bug: a decentralized issue tracker

[![Build Status][ci/badge]][ci/url]

[![Backers on Open Collective][backers/badge]][oc]
[![Sponsors on Open Collective][sponsors/badge]][oc]
[![GPL v3 License][license/badge]][license/url]
[![GoDoc][godoc/badge]][godoc/url]
[![Go Report Card][report-card/badge]][report-card/url]
[![Matrix][matrix/badge]][matrix/url]

[Issues] - [Documentation][doc] - [Feedback][discuss]

</div>

## Overview<a name="overview"></a>

[git-bug](https://github.com/git-bug/git-bug) is a standalone, distributed,
offline-first issue management tool that embeds issues, comments, and more as
objects in a git repository (_not files!_), enabling you to push and pull them
to one or more remotes.

<!-- mdformat-toc start --slug=github --maxlevel=6 --minlevel=2 -->

- [Overview](#overview)
- [Key Features](#key-features)
- [Getting Started](#getting-started)
- [Our Community](#our-community)
  - [Backers :star:](#backers-star)
  - [Sponsors :star2:](#sponsors-star2)
- [License](#license)

<!-- mdformat-toc end -->

## Key Features<a name="key-features"></a>

- **Native Git Storage:** Manage issues, users, and comments directly within
  your repository - keeping everything versioned and clutter-free
- **Distributed & Versioned:** Leverage Git’s decentralized architecture to work
  offline and sync seamlessly later
- **Lightning Fast:** List and search issues in mere _milliseconds_
- **Third-Party Bridges:** Easily synchronize issues with platforms like GitHub
  and GitLab [using bridges][doc/usage/bridges]
- **Flexible Interfaces:** Choose how you interact - via CLI, TUI, or a web
  browser
- **Effortless Integration:** Start managing issues your repository with minimal
  setup

## Getting Started<a name="getting-started"></a>

- :triangular_flag_on_post: **Install:** Check out
  [`//:INSTALLATION.md`][doc/install] for step-by-step installation instructions
  or explore [the latest release][rel/latest] to get started immediately.
- :page_with_curl: **Explore:** Read [the documentation][doc] to learn how to
  use `git-bug` effectively
- :computer: **Contribute:** Interested in hacking on `git-bug`? Head over to
  [`//:CONTRIBUTING.md`][contrib] and see how you can help shape the project
- :speech_balloon: **Connect:** Chat with us live on Matrix at
  [`#git-bug:matrix.org`][matrix/url]
- :books: **Discuss:** Browse [existing discussions][discuss] or
  [start a new one][discuss/new] to ask questions and share ideas

## Our Community<a name="our-community"></a>

`git-bug` thrives thanks to the passion of its [contributors], the generosity of
independent [backers][oc], and the strategic support of our [sponsors][oc]. Each
of you plays a crucial role in our journey, and we deeply appreciate every
contribution that helps drive our project forward.

_[Make a contribution][oc] to support this project and get featured below!_

### Backers :star:<a name="backers-star"></a>

<div align="center">

[![backers][backers/image]][oc]

</div>

### Sponsors :star2:<a name="sponsors-star2"></a>

<div align="center">

[![][sponsor/0]][sponsor/0/url]

</div>

## License<a name="license"></a>

Unless otherwise stated, this project and all assets within it are released
under [GPLv3][license/url] or later, copyright [Michael Muré][gh/mm].

The `git-bug` logo is authored by and copyright [Viktor Teplov][gh/vandesign],
released under a [CC BY 4.0][license/logo] license.

______________________________________________________________________

<div align="center">

This project and its vibrant community was initially dreamt up and built by
[Michael Muré][gh/mm].

Thank you for all of your hard work!

:heart:

</div>

[backers/badge]: https://opencollective.com/git-bug/backers/badge.svg
[backers/image]: https://opencollective.com/git-bug/tiers/backer.svg?avatarHeight=50
[ci/badge]: https://github.com/git-bug/git-bug/actions/workflows/trunk.yml/badge.svg
[ci/url]: https://github.com/git-bug/git-bug/actions/workflows/trunk.yml
[contrib]: ./CONTRIBUTING.md
[contributors]: https://github.com/git-bug/git-bug/graphs/contributors
[discuss]: https://github.com/git-bug/git-bug/discussions
[discuss/new]: https://github.com/git-bug/git-bug/discussions/new/choose
[doc]: ./doc
[doc/install]: ./INSTALLATION.md
[doc/usage/bridges]: ./doc/usage/third-party.md
[gh/mm]: https://github.com/MichaelMure
[gh/vandesign]: https://github.com/vandesign
[godoc/badge]: https://godoc.org/github.com/git-bug/git-bug?status.svg
[godoc/url]: https://godoc.org/github.com/git-bug/git-bug
[issues]: https://github.com/git-bug/git-bug/issues
[license/badge]: https://img.shields.io/badge/License-GPLv3+-blue.svg
[license/logo]: ./misc/logo/LICENSE
[license/url]: ./LICENSE
[matrix/badge]: https://img.shields.io/badge/chat%20on%20matrix-%23238636
[matrix/url]: https://matrix.to/#/#git-bug:matrix.org
[oc]: https://opencollective.com/git-bug
[rel/latest]: https://github.com/git-bug/git-bug/releases/latest
[report-card/badge]: https://goreportcard.com/badge/github.com/git-bug/git-bug
[report-card/url]: https://goreportcard.com/report/github.com/git-bug/git-bug
[sponsor/0]: https://opencollective.com/git-bug/tiers/sponsor/0/avatar.svg
[sponsor/0/url]: https://opencollective.com/git-bug/sponsor/0/website
[sponsors/badge]: https://opencollective.com/git-bug/sponsors/badge.svg
