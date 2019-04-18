# Internal architecture

This documentation only provide a bird-eye view of git-bug's internals. For more details, you should read the other documentations and the various comment/documentations scattered in the codebase.

## Overview 

```
                                                    .--------------.
    |                                               |    webui     |   |
    |                                               '--------------'   |
    |   .---------------..-----------..------------..--------------.   |
    |   |   commands    ||  bridge   ||   termui   ||   graphql    |   |
    |   '---------------''-----------''------------''--------------'   |
    |   .----------------------------------------------------------.   |
    |   |                          cache                           |   |
    |   |----------------------------------------------------------|   |
    |   | BugCache,BugExcerpt,IdentityCache,IdentityExcerpt        |   |
    |   |                                                          |   |
    |   '----------------------------------------------------------'   |
    |   .-----------------------------..---------------------------.   |
    |   |             bug             ||         identity          |   |
    |   |-----------------------------||---------------------------|   |
    |   | Bug,Operation,Snapshot      || Identity,Version          |   |
    |   '-----------------------------''---------------------------'   |
    |   .----------------------------------------------------------.   |
    v   |                        repository                        |   v
        '----------------------------------------------------------'
```

Here is the internal architecture of git-bug. As you can see, it's a layered architecture.

As a general rule of thumbs, each layer use the directly underlying layer to access and interact with the data. As an example, the `commands` package will not directly use the `bug` or `repository` package. It will request the data from the `cache` layer and go from there. Of course, the `commands` package will ultimately use types defined in the lower level package like `Bug`, but retrieving and changing the data has to go through the `cache` layer to ensure that bugs are properly deduplicated in memory.

## repository

The package `repository` implement the interaction with the git repository on disk. 

A series of interfaces (`RepoCommon`, `Repo` and `ClockedRepo`) define convenient for our usage access and manipulation methods for the data stored in git.

Those interfaces are implemented by `GitRepo` as well as a mock for testing.

## identity

The package `identity` contains the identity data model and the related low-level functions.

In particular, this package contains:
- `Identity`, the fully-featured identity, holding a series of `Version` stored in its dedicated structure in git
- `Bare`, the simple legacy identity, stored directly in a bug `Operation`

## bug

The package `bug` contains the bug data model and the related low-level functions.

In particular, this package contains:
- `Operation`, the interface to fulfill for an edit operation of a Bug
- `OpBase`, implementing the common code for all operations
- `OperationPack`, an array of `Operation`
- `Bug`, holding all the data of a bug
- `OperationIterator`, allowing to easily iterate over the operation of a bug
- all the existing operations (Create, AddComment, SetTitle ...)
- `Snapshot`, holding a compiled version of a bug

## cache

The package `cache` implements a caching layer on top of the low-level `bug` and `identity`package to provide efficient querying, filtering, sorting.

This cache main function is to provide some guarantee and features for the upper layers:
1. After being loaded, a Bug is kept in memory in the cache, allowing for fast access later.
2. The cache maintain in memory and on disk a pre-digested excerpt for each bug, allowing for fast querying the whole set of bugs without having to load them individually.
3. The cache guarantee that a single instance of a Bug is loaded at once, avoiding loss of data that we could have with multiple copies in the same process.
4. The same way, the cache maintain in memory a single copy of the loaded identities.

The cache also protect the on-disk data by locking the git repository for its own usage, by writing a lock file. Of course, normal git operations are not affected, only git-bug related one.

In particular, this package contains:
- `BugCache`, wrapping a `Bug` in a cached version in memory, maintaining efficiently a `Snapshot` and providing a simplified API
- `BugExcerpt`, holding a small subset of data for each bug, allowing for a very fast indexing, filtering, sorting and querying
- `IdentityCache`, wrapping an `Identity` in a cached version in memory and providing a simplified API
- `IdentityExcerpt`, holding a small subset of data for each identity, allowing for a very fast indexing, filtering, sorting and querying.
- `Query` and a series of `Filter` to implement the query language

## commands

The package `commands` contains all the CLI commands and subcommands, implemented with the [cobra](https://github.com/spf13/cobra) library. Thanks to this library, bash and zsh completion, manpages and markdown documentation are automatically generated.

## termui

The package `termui` contains the interactive terminal user interface, implemented with the [gocui](https://github.com/jroimartin/gocui) library.

## graphql

The package `graphql` implement the GraphQL API, mapping the data model and providing read/write access from outside of the process. This API is in particular used by the webUI but could be used to implement other user interfaces or bridges with other systems.

## webui

The package `webui` hold the web based user interface, implemented in both go and javascript.

The javascript code is compiled and packaged inside the go binary, allowing for a single file distribution of git-bug.

When the webUI is started from the CLI command, a localhost HTTP server is started to serve the webUI resources (html, js, css), as well as the GraphQL API. When the webUI is loaded in the browser, it interact with the git-bug process through the GraphQL API to load and edit bugs.

## bridge

The package `bridge` contains the various bridge implementation with other external bug trackers.
