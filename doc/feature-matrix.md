# User facing capabilities

This document tries to give an overview of what is currently supported, and by
extension where effort can be focused to bring feature completion and parity.

As git-bug is a free software project, accept and rely on contributor, those
feature matrices kinda define a roadmap, in the sense that anything mentioned
below is a planned feature and can be worked on. This does not mean that a
feature not mentioned here should not be considered, just maybe check the issue
tracker and come talk about it.

This document however does not show all the untold work required to support
those user-facing capabilities. There has been a ton of work behind the scene
and more will be required over time.

✅: working 🟠: partial implementation ❌: not working

## Other goals

Some goals don't really fit below, so I'll mention them here:

- have the webUI accept external OAuth (Github, ...) and act as a public portal
  where user outside the project can browse and interact with the project
- project configuration (valid labels, ...)
- commit signature to fully authenticate user's interaction
- interface with the system keyring, to distribute and expose known public keys
  and allow checking signed commit in normal git workflow
- privileged roles (admin, ...) and enforcing the corresponding rules
- package the webui as a desktop app

Additionally, some other are captured as
[Github issues](https://github.com/git-bug/git-bug/issues) or
[Discussions](https://github.com/git-bug/git-bug/discussions).

## Entities

The most high level overview of what kind of entities are supported and where.

|                | Core | CLI | TermUI | WebUI |
| -------------- | :--: | :-: | :----: | :---: |
| Identities     |  ✅  | ✅  |   ✅   |  ✅   |
| Bug            |  ✅  | ✅  |   ✅   |  ✅   |
| Board          |  🟠  | 🟠  |   ❌   |  ❌   |
| Pull-request   |  ❌  | ❌  |   ❌   |  ❌   |
| Project Config |  ❌  | ❌  |   ❌   |  ❌   |

More specific features across the board.

|                    | Core | CLI | TermUI | WebUI |
| ------------------ | :--: | :-: | :----: | :---: |
| Media embedding    |  🟠  | ❌  |   ❌   |  ❌   |
| Fast indexing      |  ✅  | ✅  |   ✅   |  ✅   |
| Markdown rendering | N/A  | ❌  |   ❌   |  ✅   |

#### Identities

|                         | Core | CLI | TermUI | WebUI |
| ----------------------- | :--: | :-: | :----: | :---: |
| Public keys             |  🟠  | ❌  |   ❌   |  ❌   |
| Private keys management |  🟠  | ❌  |   ❌   |  ❌   |
| Identity edition        |  ✅  | ✅  |   ❌   |  ❌   |
| Identity adoption       |  ✅  | ✅  |   ❌   |  ❌   |
| Identity protection     |  🟠  | ❌  |   ❌   |  ❌   |

#### Bugs

|                   | Core | CLI | TermUI | WebUI |
| ----------------- | :--: | :-: | :----: | :---: |
| Comments          |  ✅  | ✅  |   ✅   |  ✅   |
| Comments edition  |  ✅  | ✅  |   ✅   |  ✅   |
| Comments deletion |  ✅  | ❌  |   ❌   |  ❌   |
| Labels            |  ✅  | ✅  |   ✅   |  ✅   |
| Status            |  ✅  | ✅  |   ✅   |  ✅   |
| Title edition     |  ✅  | ✅  |   ✅   |  ✅   |
| Assignee          |  ❌  | ❌  |   ❌   |  ❌   |
| Milestone         |  ❌  | ❌  |   ❌   |  ❌   |

## Bridges

### Importers

General capabilities of importers:

|                                                 | Github | Gitlab | Jira | Launchpad | Gitea |
| ----------------------------------------------- | :----: | :----: | :--: | :-------: | :---: |
| **incremental**<br/>(can import more than once) |   ✅   |   ✅   |  ✅  |    ❌     |  🟠   |
| **with resume**<br/>(download only new data)    |   ✅   |   ✅   |  ✅  |    ❌     |  ❌   |
| **media/files**                                 |   ❌   |   ❌   |  ❌  |    ❌     |  ❌   |
| **automated test suite**                        |   ✅   |   ✅   |  ❌  |    ❌     |  ❌   |

Identity support:

|                   | Github | Gitlab | Jira | Launchpad | Gitea |
| ----------------- | :----: | :----: | :--: | :-------: | :---: |
| **identities**    |   ✅   |   ✅   |  ✅  |    ✅     |  🟠   |
| identities update |   ❌   |   ❌   |  ❌  |    ❌     |  ❌   |
| public keys       |   ❌   |   ❌   |  ❌  |    ❌     |  ❌   |

Bug support:

|                  | Github | Gitlab | Jira | Launchpad | Gitea |
| ---------------- | :----: | :----: | :--: | :-------: | :---: |
| **bug**          |   ✅   |   ✅   |  ✅  |    ✅     |  🟠   |
| comments         |   ✅   |   ✅   |  ✅  |    ✅     |  ❌   |
| comment editions |   ✅   |   ❌   |  ✅  |    ❌     |  ❌   |
| labels           |   ✅   |   ✅   |  ✅  |    ❌     |  ❌   |
| status           |   ✅   |   ✅   |  ✅  |    ❌     |  ❌   |
| title edition    |   ✅   |   ✅   |  ✅  |    ❌     |  ❌   |
| Assignee         |   ❌   |   ❌   |  ❌  |    ❌     |  ❌   |
| Milestone        |   ❌   |   ❌   |  ❌  |    ❌     |  ❌   |

Board support:

|           | Github | Gitlab | Jira | Launchpad | Gitea |
| --------- | :----: | :----: | :--: | :-------: | :---: |
| **board** |   ❌   |   ❌   |  ❌  |    ❌     |  ❌   |

### Exporters

**General capabilities of exporters**:

|                                                 | Github | Gitlab | Jira | Gitea |
| ----------------------------------------------- | :----: | :----: | :--: | :---: |
| **incremental**<br/>(can export more than once) |   ✅   |   ✅   |  ✅  |  ❌   |
| **with resume**<br/>(upload only new data)      |   ✅   |   ✅   |  ✅  |  ❌   |
| **automated test suite**                        |   ✅   |   ✅   |  ❌  |  ❌   |

**Identity support**:

|                   | Github | Gitlab | Jira | Gitea |
| ----------------- | :----: | :----: | :--: | :---: |
| **identities**    |   ✅   |   ✅   |  ✅  |   ❌  |
| identities update |   ❌   |   ❌   |  ❌  |   ❌  |

Note: as the target bug tracker require accounts and credentials, there is only
so much that an exporter can do about identities. A bridge should be able to
load and use credentials for multiple remote account, but when they are not
available, the corresponding changes can't be replicated.

**Bug support**:

|                  | Github | Gitlab | Jira | Gitea |
| ---------------- | :----: | :----: | :--: | :---: |
| **bugs**         |   ✅   |   ✅   |  ✅  |  ❌   |
| comments         |   ✅   |   ✅   |  ✅  |  ❌   |
| comment editions |   ✅   |   ✅   |  ✅  |  ❌   |
| labels           |   ✅   |   ✅   |  ✅  |  ❌   |
| status           |   ✅   |   ✅   |  ✅  |  ❌   |
| title edition    |   ✅   |   ✅   |  ✅  |  ❌   |
| Assignee         |   ❌   |   ❌   |  ❌  |  ❌   |
| Milestone        |   ❌   |   ❌   |  ❌  |  ❌   |
