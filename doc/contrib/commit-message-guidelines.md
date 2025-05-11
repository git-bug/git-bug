# Commit message guidelines

We have very strict rules over how commit messages must be formatted. This
format leads to easier to read commit history, and makes it possible to generate
a changelog for releases.

<!-- mdformat-toc start --slug=github --maxlevel=4 --minlevel=2 -->

- [Specification](#specification)
  - [Type (required)](#type-required)
  - [Scope (optional)](#scope-optional)
  - [Subject (required)](#subject-required)
  - [Body (optional)](#body-optional)
  - [Footer (optional, conditionally required)](#footer-optional-conditionally-required)
    - [Breaking changes](#breaking-changes)

<!-- mdformat-toc end -->

## Specification<a name="specification"></a>

Commit subjects and messages should follow the conventional commit message
specification [version `1.0.0`][cc-1.0.0].

```
<type>[optional scope][!]: <subject>

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

If the scope is provided, it should be the app or package name as it would be
perceived by a person reading the changelog.

The following scopes are supported:

- `api`
- `bridge`
- `bridge/github`
- `bridge/gitlab`
- `bridge/jira`
- `bugs`
- `cli`
- `config`
- `core`
- `git`
- `tui`
- `web`

The following additional _special scopes_, which do not relate to any internal
package, are supported:

- `changelog` used for changes to `//:CHANGELOG.md` and changelog generation
- `dev-infra` used for changes under `//tools` or dev shell configuration

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

### Footer (optional, conditionally required)<a name="footer-optional-conditionally-required"></a>

The footer should contain any information about **breaking changes** and is also
the place to reference issues that the change closes.

#### Breaking changes<a name="breaking-changes"></a>

To indicate that a commit introduces a breaking change, append `!` after the
type or scope. You can optionally provide additional information (for example,
migration instructions) by adding a `BREAKING-CHANGE` trailer to the footer.
This additional information will be shown in the changelog.

> [!NOTE]
> Breaking changes in this repository **always require** the `!` suffix.

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
