# git-bug

> Bugtracker embedded in Git 

## Install

```shell
go get github.com/MichaelMure/git-bug
```

If it's not done already, add golang binary directory in your PATH:

```bash
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

That's all !

## Usage

It's really a WIP but you can already create a bug:

```
git bug new "This doesn't even build"
```

Your favorite editor will open to write a description.

You can push your new entry to a remote:
```
git bug push [<remote>]
```

And pull for updates:
```
git bug pull [<remote>]
```

## Contribute

PRs accepted.

## License


GPLv3 © Michael Muré
