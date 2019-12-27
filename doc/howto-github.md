# How-to: Read/write offline Github issues with git-bug

[git-bug](https://github.com/MichaelMure/git-bug) is a standalone distributed bug-tracker that is embedded in git. In short, if you have a git repository you can use it to store bugs alongside your code (without mixing them though!), push and pull them to/from a normal git remote to collaborate.

Bridges with other bug-trackers are first-class citizen in `git-bug`. Notably, they are bidirectional, incremental and relatively fast. This means that a perfectly valid way to use `git-bug` is as a sort of remote for Github where you synchronize all the issues of a repository to later read and edit them and then propagate your changes back to Github.

This has several upsides:
- works offline, including edition
- browsing is pretty much instant
- you get to choose the UI you prefer between CLI, interactive terminal UI or web UI
- you get a near complete backup in case Github is down or no longer fit your needs

## Installation

Follow the [installation instruction](https://github.com/MichaelMure/git-bug#install). The simplest way is to download a pre-compiled binary from [the latest release](https://github.com/MichaelMure/git-bug/releases/latest) and to put it anywhere in your `$PATH`. 

Check that `git-bug` is properly installed by running `git bug version`. If everything is alright, the version of the binary will be displayed.

## Configuration

1. From within the git repository you care about, run `git bug bridge configure` and follow the wizard's steps:
    1. Choose `github`.
    1. Type a name for the bridge configuration. As you can configure multiple bridges, this name will allow you to choose when there is an ambiguity.
    1. Setup the remote Github project. The wizard is smart enough to inspect the git remote and detect the potential project. Otherwise, enter the project URL like this: `https://github.com/MichaelMure/git-bug`
    1. Setup an authentication token. You can either use the interactive token creation, enter your own token or select an existing token, if any.
1. Run `git bug bridge pull` and let it run to import the issues and identities.
1. Find your imported identity by running `git bug user ls` and select it as your own with `git bug user adopt <id>`

## Basic usage

You can interact with `git-bug` through the command line (see the [Readme](../README.md#cli-usage) for more details):
```bash
# Create a new bug
git bug add
# List existing bugs
git bug ls
# Display a bug's detail
git bug show <bugId>
# Add a new comment
git bug comment <bugId>
# Push everything to a normal git remote
git bug push [<remote>]
# Pull updates from a git remote
git bug pull [<remote>]
```

In particular, the key commands to interact with Github are:
```bash
# Replicate your changes to the remote bug-tracker
git bug bridge push [<bridge>]
# Retrieve updates from the remote bug-tracker
git bug bridge pull [<bridge>]
```

The command line tools are really meant for programmatic usage or to integrate `git-bug` into your editor of choice. For day to day usage, the recommended way is the interactive terminal UI. You can start it with `git bug termui`:

![termui recording](../misc/termui_recording.gif)

For a richer and more user friendly UI, `git-bug` proposes a web UI (read-only at the moment). You can start it with `git bug webui`:

![web UI screenshot](../misc/webui2.png)
