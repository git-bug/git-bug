# git-bug

> Bugtracker embedded in Git

Would it be nice to not have to rely on a web service somewhere to deal with bugs ?

Would it be nice to be able to browse and edit bug report offline ?

`git-bug` is a bugtracker embedded in `git`. It use the same internal storage so it doesn't pollute your project. As you would do with commits and branches, you can push your bugs to the same git remote your are already using to collaborate with other peoples.

:construction: This is for now a proof of concept. Expect dragons and unfinished business. :construction:

## Install

```shell
go get github.com/MichaelMure/git-bug
```

If it's not done already, add golang binary directory in your PATH:

```bash
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

That's all ! In the future, pre-compiled binary will be provided for convenience.

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

You can now use commands like `show`, `comment`, `open` or `close` to display and modify bugs.

## All commands

```bash
# Mark the bug as closed
git bug close <id>

# Display available commands
git bug commands [<option>...]

# Add a new comment to a bug
git bug comment [<options>...] <id>

# Manipulate bug's label
git bug label <id> [<option>...] [<label>...]

# Display a summary of all bugs
git bug ls 

# Create a new bug
git bug new [<option>...] <title>

# Mark the bug as open
git bug open <id>

# Pull bugs update from a git remote
git bug pull [<remote>]

# Push bugs update to a git remote
git bug push [<remote>]

# Display the details of a bug
git bug show <id>

# Launch the web UI
git bug webui 
```

## Internals

Interested by how it works ? Have a look at the [data model](doc/model.md).

## Planned features

- interactive CLI UI
- rich web UI
- media embedding
- import/export of github issue
- inflatable raptor

## Contribute

PRs accepted.

## License


GPLv3 or later © Michael Muré
