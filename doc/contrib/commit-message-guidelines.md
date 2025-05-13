# Commit message guidelines

We have very strict rules over how commit messages must be formatted. This
format leads to easier to read commit history, and makes it possible to generate
a changelog for releases.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Specification](#specification)
  - [Format overview](#format-overview)
  - [Type (required)](#type-required)
  - [Scope (optional)](#scope-optional)
  - [Subject (required)](#subject-required)
  - [Body (optional)](#body-optional)
    - [Anchors / Hyperlinks](#anchors--hyperlinks)
  - [Footer (optional, conditionally required)](#footer-optional-conditionally-required)
    - [Breaking changes](#breaking-changes)

<!-- mdformat-toc end -->

## Specification<a name="specification"></a>

Commit subjects and messages should follow the format specified below, which is
a superset of the conventional commits specification
[version `1.0.0`][cc-1.0.0].

**A superset? What's different?**

We require just one small change from the spec:

- Appending the type and scope with `!` is _always_ required for breaking
  changes. This is done by improve visibility of breaking changes in the commit
  log.

### Format overview<a name="format-overview"></a>

```
<type>[scope][!]: <subject>

[optional body]

[optional footer(s)]
```

### Type (required)<a name="type-required"></a>

Valid values for `type` MUST be one of the following:

| Type       | Description                                          |
| ---------- | ---------------------------------------------------- |
| `build`    | Changes that affect the build system or dependencies |
| `ci`       | Changes to the CI configuration files                |
| `docs`     | Changes consisting only of documentation changes     |
| `feat`     | A new feature                                        |
| `fix`      | A bug fix                                            |
| `perf`     | Changes that improve performance                     |
| `refactor` | A change that neither fixes a bug or adds a feature  |
| `test`     | Adding missing tests or correcting existing tests    |

### Scope (optional)<a name="scope-optional"></a>

The scope should be the app or package name as it would be perceived by a person
reading the changelog.

The following scopes are supported:

- `api`
- `bridge`
- `bridge/github`
- `bridge/gitlab`
- `bridge/jira`
- `bugs`
- `cache`
- `cli`
- `config`
- `git`
- `tui`
- `web`

There are a few exceptions to the "use the package name" rule:

- `changelog`: used for changes related to the changelog or its generation
- `dev-infra`: used for developer-centric changes (docs, scripts, tools, etc)
- `treewide`: used for repo-wide changes (e.g. bulk formatting, refactoring)
- _none / unset_: useful for `build`, `ci`, and `test` (if changing a test lib,
  or multiple tests)

### Subject (required)<a name="subject-required"></a>

The subject should contain a succinct, descriptive summary of the change.

- Use the imperative, present tense: "change" not "changed" or "changes"
- Do not use capital letters, except in acronyms (like `HTTP`)
- Do not end the subject with a `.` or `?` or any similar end-of-sentence symbol

### Body (optional)<a name="body-optional"></a>

The body is used to provide additional context: the _what_, _why_, and _how_ for
the change. Just as in the **subject**, use the imperative tense. The body
should contain the motivation for the change, and contrast it with the previous
behavior.

#### Anchors / Hyperlinks<a name="anchors--hyperlinks"></a>

If you include anchors, otherwise known as hyperlinks (or just "links") in the
body, be sure to use the _reference style_ for anchor links.

**Incorrect - do not use inline links**

```
Lorem ipsum dolar [sit amet](https://foo.com), consectetur adipiscing elit. In
hendrerit orci et risus vehicula venenatis.
```

**Correct - use reference style links**

Both of the below examples are valid.

```
Lorem ipsum dolar [sit amet][0], consectetur adipiscing elit. In hendrerit orci
et risus vehicula venenatis.

[0]: https://foo.com
```

```
Lorem ipsum dolar sit amet [0], consectetur adipiscing elit. In hendrerit orci
et risus vehicula venenatis.

[0]: https://foo.com
```

### Footer (optional, conditionally required)<a name="footer-optional-conditionally-required"></a>

The footer should contain any information about **breaking changes** (see below)
and is also the place to reference issues that the change closes, if any.

**Examples**

```
Closes: #000
Closes: git-bug/git-bug#000
Fixes: #000
Ref: https://domain.com/some-page/foo/bar
Ref: https://github.com/git-bug/git-bug/discussions/000
```

#### Breaking changes<a name="breaking-changes"></a>

To indicate that a commit introduces a breaking change, append `!` after the
type and scope (**this is required**). You can optionally provide additional
information (for example, migration instructions) by adding a `BREAKING-CHANGE`
trailer to the footer. This additional information will be shown in the
changelog.

> [!NOTE]
> Breaking changes in this repository **always require** the `!` suffix in order
> to improve visibility of breaking changes in the commit log.

**Examples**

Below are commit message examples you can use as references for writing a commit
message for a breaking change.

**Unscoped breaking change without additional information**

```
feat!: remove the ABC bridge
```

**Scoped breaking change without additional information**

```
feat(config)!: remove configuration option: foo.bar
```

**Scoped breaking change with additional information**

```
feat(config)!: remove option: foo.bar

BREAKING-CHANGE: Users should migrate to `bar.baz`
```

**Scoped breaking change with multiple lines**

If your breaking change description spans multiple lines, be sure to indent each
subsequent line with at least one space so that the message is parsed correctly.

```
feat(config)!: remove option: foo.bar

BREAKING-CHANGE: Users should migrate to `bar.baz` in order to continue
 operating the tool and avoid a parsing error when the configuration is loaded,
 which would throw an error stating that the `foo.bar` option doesn't exist.
```

[cc-1.0.0]: https://www.conventionalcommits.org/en/v1.0.0/#specification
