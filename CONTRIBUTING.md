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

The web UI is compiled into the binary with `//go:embed`, and `webui/dist` is
not tracked in git, so **the web UI has to be built before the Go build will
succeed**. Every `make` target that produces a binary does this for you. If you
are running `go build` or `go test` by hand for the first time, run
`make build-webui` once beforehand.

To work on the web UI itself, see [`webui/README.md`](./webui/README.md).

## Checks

CI checks that the Go code is formatted (`gofmt -l`) and that the generated
files below are up to date. `make secure` checks dependencies for known
vulnerabilities.

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
