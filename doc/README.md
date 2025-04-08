# Documentation

## Usage

The documentation listed below aims to help provide insight into the usage of
`git-bug`.

- Read about the different [types of workflows](./usage/workflows.md) and check
  the [feature matrix](./feature-matrix.md) to learn about `git-bug`
- Check the [CLI documentation](./md/git-bug.md) for commands and options (or
  run `man git-bug` after [installation](../INSTALLATION.md))
- Filter results using the [query language](./usage/query-language.md)
- Learn how to [sync third party issues](./usage/third-party.md) for offline
  reading and editing

## For developers

- Read through [`//:CONTRIBUTING.md`][contrib]
- Get a [bird's-eye overview](./design/architecture.md) of the architecture
- Read about the [data model](./design/data-model.md) to gain a deeper
  understanding of the internals that comprise `git-bug`
- [`//entity/dag:example_test.go`](../entity/dag/example_test.go) is a good
  reference to learn how to create a new distributed entity
- Read the [bridge design documents](./design/bridges) to learn more about each
  bridge

[contrib]: ../CONTRIBUTING.md
