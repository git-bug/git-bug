# Contributing to git-bug

## Development environment

`git-bug` is a single Go binary with a React web UI compiled into it. You need
Go, Node.js and pnpm to build it.

The versions we develop and test against are pinned in
[`.tool-versions`](./.tool-versions). If you use [asdf][asdf], you can install
all of them in one go:

```shell
# once, to register the plugins
asdf plugin add golang
asdf plugin add nodejs
asdf plugin add pnpm
asdf plugin add goreleaser

# then, from the root of the repository
asdf install
```

`asdf` will pick the right versions up automatically whenever you are inside the
repository.

If you would rather not use `asdf`, install the same versions by hand; anything
reasonably close should work. `goreleaser` is only needed if you want to build
release artifacts locally, so feel free to skip it.

[asdf]: https://asdf-vm.com/guide/getting-started.html

## Building and testing

```shell
make            # build the web UI, then the binary, into ./git-bug
make install    # same, but install into $GOPATH/bin
make build/debug  # a debugger-friendly build (no optimisation, no inlining)
make test       # run the Go test suite
```

The web UI is compiled into the binary with `//go:embed`, but `webui/dist` is
not tracked in git, so the embed sits behind a `webui` build tag. Without the
tag the package still compiles with an empty asset set, which means `go build`,
`go test` and `go install` all work on a fresh clone with no Node.js toolchain
— you just get a binary whose `git-bug webui` command explains that it has no
web UI and how to get one.

Every `make` target that produces a binary builds the frontend and passes
`-tags webui`, so `make build` and `make install` give you the real thing.
Release binaries do the same, via [`.goreleaser.yaml`](./.goreleaser.yaml).

To work on the web UI itself, see [`webui/README.md`](./webui/README.md).

## Checks

CI checks that the Go code is formatted (`gofmt -l`), that the generated files
below are up to date, and that no dependency has a known vulnerability
reachable from our code. That last one is `make secure` locally.

The man pages in `doc/man` and the shell completions in `misc/completion` are
generated from the command tree and committed to the repository. If you add or
change a command, run `go generate` and commit the result.

For the web UI, `cd webui && pnpm run check` lints and checks formatting.

## Releases

Releases are cut by [GoReleaser](https://goreleaser.com) when a `v*` tag is
pushed; see [`.goreleaser.yaml`](./.goreleaser.yaml) and
[`.github/workflows/release.yml`](./.github/workflows/release.yml).

To see what a release would produce without publishing anything, run
`goreleaser release --snapshot --clean`; it writes the artifacts into `dist/`.
