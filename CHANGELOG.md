# Changelog

All notable changes to the project will be documented in this file. It is
non-exhaustive by design, and only contains public-facing application and API
changes. Internal, developer-centric changes can be seen by looking at the
commit log.

## 0.10.1 (2025-05-19)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline v0.10.0..v0.10.1
```

### Bug fixes

- **cli**: ignore missing sections when removing configuration (ddb22a2f)

## 0.10.0 (2025-05-18)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline v0.10.0
```

### Documentation

- **bridge**: correct command used to create a new bridge (9942337b)

### Features

- **web**: simplify header navigation (7e95b169)
- **webui**: remark upgrade + gfm + syntax highlighting (6ee47b96)

### Feat

- **BREAKING CHANGE**: **dev-infra**: remove gokart (89b880bd)

## 0.10.0 (2025-05-18)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline v0.9.0..v0.10.0
```

### Documentation

- **bridge**: correct command used to create a new bridge (9942337b)

### Features

- **web**: simplify header navigation (7e95b169)
- **web**: remark upgrade + gfm + syntax highlighting (6ee47b96)

## 0.9.0 (2025-05-12)

This release contains minor improvements and bug fixes.

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline v0.8.1..v0.9.0
```

### Bug fixes

- **completion**: remove errata from string literal (aa102c91)

### Features

- **tui**: improve readability of the help bar (23be684a)

## 0.8.1 (2025-05-05)

This release contains the culmination of new features, bug fixes, and other
miscellaneous changes (documentation, tooling) since the last release in 2022.

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline v0.8.0..v0.8.1
```

### Bug fixes

- remove repeated use of the same fmt.Errorf() calls (0cd2f3b4)
- use prerelease of GoKart with repaired panic (344438b9)
- keyrings must return keys with entities/identities (de6f5404)
- resolve Go vulnerabilities (33e3e4b6)
- (cli): run tests in ./commands/... without ANSI color (e4707cd8)
- (cli): create env.Env once for all Cobra commands (0bddfe1d)
- (cli): replace missing import (723b3c41)
- parse submodule .git files instead of erroring (e97df9c8)
- openpgp handling to sign/check (429b913d)
- correct typo: acceps => accepts (76de669d)
- bump to go v1.22.5 (f79ea38c)
- add missing `with` property to //.github/workflows:cron.yml (eef62798)
- add write for prs: stale/issue-and-pr (6c9aade8)
- move codeql into an independent workflow (1fa858dc)
- run the presubmit pipeline for PRs (5893f948)
- correct path for reusable workflow: lifecycle (1dd81071)
- typos in docs (d499b6e9)
- set GitLastTag to an empty string when git-describe errors (25f755cb)
- refactor how gitlab title changes are detected (197eb599)
- use correct url for gitlab PATs (7b6eb5db)
- use -0700 when formatting time (edbd105c)
- checkout repo before setting up go environment (5e8efbae)
- resolve the remote URI using url.\*.insteadOf (a150cdb0)

### Documentation

- normalize verb tense and fix typo (8537869a)
- add a feature matrix (3c1b8fd0)
- update install, contrib, and usage documentation (96c7a111)

### Features

- wrap ErrNoConfigEntry to report missing key (64c18b15)
- wrap ErrMultipleConfigEntry to report duplicate key (49929c03)
- upgrade go-git to v5.1.1 (7c4a3b12)
- detect os.Stdin/os.Stdout mode (14603773)
- use isatty to detect a Termios instead (a7364015)
- add concurrency limits to all pipelines (a4b88586)
- update action library versions (eda0d672)
- add initial nix development shell (bf753031)
- add a commit message template (825eecef)
- add a common file for git-blame ignored revisions (4089b169)
- add initial editorconfig configuration file (be005f6a)
- add workflow for triaging stale issues and prs (00f5265a)
- increase operations per run for workflow: cron (c67d75fa)
- allow for manual execution of workflow: cron (ea86d570)
- refactor pipelines into reusable workflows (5eabe549)
- bump node versions to 16.x, 18.x, and 20.x (7918af66)
- improved lifecycle management with stale-bot (91fa676c)
- merge go directive and toolchain specification (66106f50)
- add package to dev shell: delve (0c0228d3)
- update references to the git-bug organization (2004fa79)
- support new exclusion label: lifecycle/pinned (f81a71a3)
- remove lifecycle/frozen (4f97349f)
- add action: auto-label (c3ab18db)
- bump to go v1.24.2 (73122def)

### Other changes

- reorg into different packages (acc9a6f3)
- add a workflow to continuously run benchmarks (c227f2e9)
- make it work? (c6bb6b9c)
- cleanup test token when test is done (10851853)
- proper reduced interface for full-text indexing (60d40d60)
- return specific error on object not found, accept multiple namespace to
  push/pull (905c9a90)
- tie up the refactor up to compiling (9b98fc06)
- generic withSnapshot, some cleanup (d65e8837)
- fix some bugs after refactor (95911100)
- tie the last printf in an event to make the core print free (13a7a599)
- move bug specific input code into commands/bug/input (d5b07f48)
- simplify cache building events handling (b2795875)
- generic `select` code, move bug completion in bugcmd (e9209878)
- don't double build the lamport clocks (c9009b52)
- remove lint security step as it's crashing (57f328fb)
- share JSON creation (5844dd0a)
- fix tests? (70b0c5b8)
- check error when closing a repo in tests (2664332b)
- temporary use a fork of go-git due to
  https://github.com/go-git/go-git/pull/659 (03dcd7ee)
- don't forget to close a file (5bf274e6)
- add a nice terminal progress bar when building the cache (7df34aa7)
- move terminal detection to Out, introduce the compagnion In (f011452a)
- adapt the output of the bug list to the terminal size (9fc8dbf4)
- remove compact style for `bug`, as the width adaptive default renderer cover
  that usage (f23a7f07)
- different pattern to detect changed flags (3e41812d)
- code cleanup, fix some edge cases (5238d1dd)
- add a helper to generate testing regex for CLI output (b66d467a)
- clean up linter complaints (cf382e0f)
- resolve PR comments (14773b16)
- faster indexing by caping Bleve batch count (f33ceb08)
- updated error message when detectGitPath fails (d9ac6583)
- improve support for gitdir indirection (27c96a40)
- fix how security tools are setup and launched (44771523)
- ignore spelling mistake in repo to be imported from github (a9697c7a)
- also teardown cleanly on SIGTERM (42aea2cd)
- better IsRunning(pid) (4b62a945)
- fix some cache building progress bar artifact (281d4a64)
- no `with` means using codespellrc, add more opt out (d8bcd71d)
- regenerate after gqlgen upgrade (31a97380)
- more ignore (de8d2c13)
- fix some struct names in comments (ce7fd6fc)
- remove refs to deprecated io/ioutil (d4f6f273)
- update go dependencies (f5076359)
- it is `new` not `configure` command (also was missing \\) (f00e42e7)
- regenerate command completion and documentation (c3ff05f9)
- make label a common type, in a similar fashion as for status (3a4b8805)
- properly namespace Bug to make space for other entities (57e71470)
- update go-git to v5@masterupdate_mods (a987d094)
- update golang.org/x/net (15d22a22)
- gofmt simplify gitlab/export_test.go (53559429)

### Reversions

- feat: increase operations per run for workflow: cron (32972230)
- Create Dependabot config file (3f84d949)

## 0.8.0 (2022-11-20)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.7.1..v0.8.0
```

### Bug fixes

- cache not rebuilding properly (c326007d)
- github action (87a2638c)
- ListCommits implementation (27e70af2)
- go sum rebase artifacts (fb9170e2)
- merge (1ced77af)
- issue with toggling the author (248201bc)
- issue with regex (bff9fa67)
- issue with keyDown propagation (72fc0ef7)
- regex issue (41ee97a4)
- remove extra mutex lock when resolving bug prefix (eda312f9)
- replace Windows line terminators (cd1099aa)
- remove only t.Parallel() (da9f95e4)
- simplify handling of Windows line terminations (1a504e05)
- remove duplication stderr/stdout set-up (848f7253)
- revert to older test harness (870fe693)
- remove obsolete test logging (2c2c4491)
- merge in CombinedId from 664 (ff1b7448)
- normalize Windows line endings -> \*nix (0f885d4f)
- normalize Windows line endings -> \*nix (golden files) (c4a4d457)
- hide tools versioning behind build tags (1dcdee49)
- correct name for one of the security phonies (8bd98454)
- process unused (but assigned) error (fc444915)
- scan PRs for insecure practices (2b47003f)

### Documentation

- fix typos (ff0ff863)
- generate concurrently (7f87eb86)
- cleanup query documentation (10a259b6)
- add missing file (d0e65d5a)
- more discoverable docs (1c219f67)
- tiny tweaks (b43a447a)
- more tiny fixes (c6be0588)
- more tiny fixes (2ade8fb1)
- add compact to docs and bash for ls command's format flag (d3f2fb0d)
- fix incorrect indentation (55a2e8e4)

### Features

- use author to filter the list (54c5b662)
- add filter by label (31871f29)
- check if there are labels (7a7e93c9)
- multiple label filter (92ce861f)
- use predefined filters (f82071a3)
- Github bridge mutation rate limit (247e1a86)
- make local storage configurable (b42fae38)
- updates default ls formatter for TSV output (a5802792)
- version tools using Go module system (d989f9b6)
- add security tools (2caade93)
- add recipes for security analysis (ec739558)
- run security checks during Go workflow (3087cdc0)

### Other changes

- document workflows (685a4fdc)
- fix image links (e43920bc)
- better phrasing (a8aecec6)
- fix a crash when trying to open a bug when there are none (8a81b9fe)
- actually test the mutator (fb31f801)
- commands/root.go: syntax (25d633d5)
- README.md: is/are (fe6e3ef4)
- make sure to disable label color escape when not on a terminal (9a00ffb7)
- update some deps (c9e4a356)
- enable Fish completion (78f39c40)
- Add support to ls dump bug information in specific formats (de5565b5)
- cleanup and re-generate files (1d06244c)
- harmonize how time are used, fix some issues in command special formats
  (aab3a04d)
- remove tie to Bug, improved and reusable testing (88ad7e60)
- more tests (939bcd57)
- render component's children as a function to avoid uncecessary rendering
  (07d6c6aa)
- pack (3aaf7758)
- refactor to avoid globals (26bd1dd1)
- open and close the backend in a single place, simplify commands (536c290d)
- merge git.Hash in for one less /util package (3cf31fc4)
- split into multiple files for readability (8a38af24)
- fix test chocking on randomized element in repo.ListRefs() (44096b78)
- fix segfault with badly loaded backend (71989045)
- minor code improvements (5c823a70)
- avoid importing a whole package to check an error (ac7e5086)
- skip the broken test as `known broken` :( (0590de9f)
- code cleanup for the rm feature (a62ce78c)
- cleanup the command's usage to avoid warnings when generating the doc
  (ae5c0967)
- fix BugExcerpt's timestamp not properly stored (92a59ece)
- make the help visually easier to parse (9ce84fc1)
- help bar background goes all the width (8eb7faf6)
- fix FreeBSD package name (e374c9da)
- use sha256 to compute labels color, to use a single hash function in the
  codebase (47ea66f6)
- fix tests (60466f86)
- simplify cache eviction (4d678f3e)
- Remove empty borders around bug table view (6824ecf0)
- pack the bug table view (5a4dc7aa)
- show the number of *additional* comments (a42abaae)
- don't pack it *that* much (807844bb)
- match the output in ls and in the termui (148b335d)
- move the random bug command on its own package (9f3a56b1)
- partial impl of a go-git backed Repo (d171e110)
- add access to the system keyring, with fallback on a file (b1274813)
- store credentials in the Keyring instead of the git config (3ecbf8db)
- some light shuffling of code (30d1640b)
- fix a todo in the gogit repo (9c1087e1)
- more go-git implementation (2bda7031)
- fix some go-git implementation (d4f1d565)
- fix gogit clock path (cdfbecf3)
- smaller interfaces (cedcc277)
- test both plain and bare, test clocks (9408f1eb)
- split Config into 2 smaller interfaces (c68be32d)
- split mocks into smaller reusable components (aa8055e2)
- only use the file backend for the keyring (c87e9aba)
- implement local/global/any config everywhere (71b7eb14)
- fix manu bugs in go-git config (4f172432)
- more config related bug fixes (eb88f0e4)
- ReadTree must accept either a commit or a tree hash (d8b49e02)
- more testing for an edge case (736d0a2f)
- implement GetCoreEditor for go-git (0acb3505)
- fix wrong ordering in gogit's ListCommit (4055495c)
- fallback editor list by looking if the binary exist (db20bc34)
- dependencies upgrades (f4433d80)
- use go-git in more places, fix push (1a0c86a1)
- fix missing keyring on the go-git repo (1eb13173)
- structural change (ca720f16)
- minor cleanup (5d1fc3ff)
- remove support for legacy identity (499dbc0a)
- updage go-git to v5.2.0 (afdbd65e)
- fix incorrect git dir on the git CLI implementation (064a96f8)
- workaround a go-git bug and ensure sorted tree object (ca4020f4)
- updage xanzy/go-gitlab to v0.38.2 (4143c3d1)
- fix edge case in git config read, affecting bridges (8a158d1f)
- upgrade spf13/cobra to v1.1.1 (86faedc9)
- expand the tokenizer/parser to parse arbitrary search terms (b285c57d)
- fix query quotation (9bea84e2)
- english specialized indexing (b494e068)
- minor cleanups (9daa8ad0)
- more work towards RepoStorage (bca9ae82)
- finish RepoStorage move (4ef2c110)
- remove the memory-only repo for now (be6e653f)
- simpler clock mutex locking (71e13032)
- move bleve there (c884d557)
- close before deleting (8128bb79)
- Pinpoint some of the reasons for bug #385 (0baf65cd)
- refactor to reuse the split function for both query and token (fab626a7)
- better powershell completion, thanks to cobra upgrade (365073d0)
- move credential loading and client creation (3d14e2e6)
- allow specifying the initial query (626ec983)
- minor fixes for the webui open with query (3a819525)
- minor code fixes (07e1c45c)
- fix eslint? (fbf7c48b)
- stay within the SPA when redirecting from the header (aeb26d0e)
- fix security issue that could lead to arbitrary code execution (9434d2ea)
- Resolve new EditComment mutation (79cc9884)
- Add EditComment to mutation type (19a68dea)
- Add EditComment mutation to schema (4960448a)
- Add EditComment input/payload to gen_models (50cd1a9c)
- Add target to EditCommentInput (cc7788ad)
- Regenerate the GraphQL-Server (2a1c7723)
- fix various config issues around case insentivity (890c014d)
- only FTS index token \< 100 characters (32958b5c)
- test for FTS bub with long description (e9856537)
- fix no-label filter not properly wired (f7dec7e9)
- match wikipedia algorithm (44d75879)
- expose all lamport clocks, move clocks in their own folder (fb0c5fd0)
- Id from data, not git + hold multiple lamport clocks (5ae8a132)
- Id from first operation data, not git + remove root link (7163b228)
- PR fixes (b01aa18d)
- unique function to generate IDs (2bf2b2d7)
- debug (497ec137)
- don't store the id in Bug, match how it's done for Identity (2788c5fc)
- fix tests (fcf43915)
- generalize the combined Ids, use 64 length (db707430)
- fix `comment edit` usage (bb8a214d)
- add error to signal invalid format (5f6a3914)
- partially add two new functions to RepoData (5c4e7de0)
- add embryo of a generic, DAG-enabled entity (9cca74cc)
- clocks and write (51ece149)
- total ordering of operations (4ef92efe)
- more progress on merging and signing (dc5059bc)
- readAll and more testing (fe4237df)
- more testing and bug fixing (e35c7c4d)
- use BFS instead of DFS to get the proper topological order (32c55a49)
- test all merge scenario (26a4b033)
- working commit signatures (2bdb1b60)
- remove the pack lamport time that doesn't bring anything actually (f7416691)
- implement remove (ef05c15f)
- expose create and edit lamport clocks (59e99811)
- clock loader (71e22d9f)
- pass the identity resolver instead of defining it once (94f06cd5)
- support different author in staging operations (99b9dd84)
- migrate to the DAG entity structure! (3f6ef508)
- make sure merge commit don't have operations (4b9862e2)
- wrap dag.Entity into a full Bug in MergeAll (45e540c1)
- no sign-post needed (bd095417)
- nonce on all operation to prevent id collision (f1d4a19a)
- add support for storing files (5215634d)
- more comments (cb9b0655)
- returning value (1216fb1e)
- many fixes following the dag entity migration (55499252)
- minor cleanups (10a80f18)
- workaround a non thread-safe path in go-git (d000838c)
- workaround more go-git concurrency issue (72197531)
- fix empty actors/participants in the index (7edb6a2c)
- attempt to fix a CI issue (7a7a4026)
- Add non-interactive option to interactive commands (1939949f)
- proper backend close on RunE error (271dc133)
- Add AddCommandAndCloseBug mutation (4043f5da)
- Add comment-and-close of a bug in one step (098bcd0c)
- Implement AddCommentAndReopenBug mutation (27b5285b)
- Add comment-and-reopen of a bug in one step (6f6831e1)
- github bridge: push then pull without duplication (476526ac)
- github import, some issue titles cause error (160ba242)
- Add new iterator with state change events (aa4e225a)
- order events on the fly (e762290e)
- re-enable previously broken test (e888391b)
- upgrade graphql-codegen dependencies (11d51bee)
- upgrade most dependencies (ce502696)
- replace React imports (bce4d095)
- upgrade react-router (b0eb041e)
- upgrade Material UI (fd17d6dd)
- replace GraphQL linter (5f35db22)
- update nodejs (03ad448c)
- fix compile (4af26663)
- fix dark theme colors (8229e80d)
- allow to resolve identities when numashalling operations (fd14a076)
- fix incorrect client creation reusing the same credential (6f112824)
- add an extensive example (450d7f7a)
- don't serialize multiple time the author, only once in OperationPack
  (c5b70d8d)
- use the correct GenBashCompletionV2 instead of the legacy function (f25690db)
- fix bash completion with `git bug` (edc8b758)
- fix incorrect query parsing with quotes escaped by the shell (b9991d84)
- lots of small ironing (3d534a70)
- strict Markdown requires empty lines before (and after) lists (33c67027)
- Adds link explaining nounce (wikipedia) (543e7b78)
- Moves example description after the example (2a0331e2)
- Links to a section further down (e652eb6f)
- Highlight some words with special meaning (00fb4bc0)
- Removes now outdated statement about ops and root (9b871c61)
- Multiple, minor readability and language improvements (75ca2ce7)
- move all completions in a dedicated folder (c732a18a)
- fix two invalid mutex lock leading to data races (fe231231)
- fix data race when closing event channel (7348fb9e)
- clean-up commented code (e29f58bf)
- close index before deleting it on disk (50de0306)
- merge in LocalStorage namespace configuration (5982e8fb)
- rearrange imports to git-bug convention (941f5b3f)
- ensure that the default repo has a non-empty name to make js/apollo happy
  (295da9c7)
- proper base operation for simplified implementation (3d454d9d)
- fix an issue where Id would be used, then changed due to metadata (d179b8b7)
- generalized resolvers to resolve any entity time when unmarshalling an
  operation (45f5f852)
- have a type for combined ids (45b04351)
- adapt to CombinedId (6ed4b8b7)
- add a flag to log handling errors (8d11e620)
- test op serialisation with the unmarshaller, to allow resolving entities
  (e1b172aa)
- update most of dependencies (c02528b7)
- put react-scripts and typescript as dev-dependency (49fe8e9f)
- better PHONY (0eef9391)
- bubble up the comment ID when created, or edited the first comment (3c6ebc2b)
- fix rate limiting (a52c474f)
- concurrent loading of clocks (d1744f5e)
- sanitize rate limit waiting time (9abeb995)
- fix incorrect loader handling (3c0fcb74)
- pack into binary (61c9f401)
- add a release workflow to build and upload binaries (c74fabd6)
- don't build for darwin/386 as support has been removed in golang (a3fa445a)

## 0.7.1 (2020-04-04)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.7.0..0.7.1
```

### Other changes

- build with go-1.14, release with go-1.13 (4096cb05)
- change title (e4f501c0)
- fix missing login in LegacyAuthorExcerpt causing panic (e0a702f4)
- add target to clean remote identities (05c968ca)
- fix bugs import url (49285b03)
- match bugs on IDs + baseURL because the URL is not stable (8389df07)
- tag bugs with the base URL, tighten the matching (43977668)
- tighten the import matching (fae3b2e7)
- tighten the bug matching (a8666bfe)
- replace the all-in-one query parser by a complete one with AST/lexer/parser
  (5e4dc87f)
- no need for an ast package (314fcbb2)
- fix a nil value access (aec81b70)
- more robust tokenizer (ecde909b)
- fix a bad login handling in the configurator (38b42bc8)
- refactor the iterator, fix bugs (f4ca533f)
- fix iterator (paginate with first index 1) and avoid the trailing API call
  (903549ca)

## 0.7.0 (2020-03-01)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.6.0..0.7.0
```

### Bug fixes

- version not set properly when built on travis (20080aa0)
- merge (20ca2bc0)
- tests ? (2e7ac569)
- usage of newIdentityRaw (d349137e)

### Documentation

- fix typos (710d8566)
- refresh the github howto (8365c633)

### Other changes

- fix edit not being pushed with baseUrl (d2ed6747)
- proper token generation URL with custom base URL (86b114ae)
- warning when the comment to be edited is missing instead of failing (ee48aef4)
- use the IntrospectionFragmentMatcher & update dependencies (5374a74e)
- custom image tag (42219ab6)
- fix column width on bug (e08ecf1a)
- open image in a new tab on click (3413ee44)
- fix width for pre tags in bug messages (e3646748)
- change primary color (f716bc1d)
- fix AppBar (8f6bc245)
- display current identity in the AppBar (def48e53)
- add logo (7de5a25f)
- remove useless conditions (70354165)
- enhance the issue list page (fa135501)
- implement filtering (4d97e3a1)
- implement issue list sort (ead5bad7)
- add open/closed issues count (adb28885)
- don't store legacy identities IDs in bug excerpt as they are not reachable.
  Fix a panic (f093be96)
- better reusable prompt functions (db893494)
- rework mutation (390b13c9)
- rework resolving of bugs, identity (da0904d2)
- make sure to have a name (8773929f)
- fix wrong error used (a335725c)
- hopefully fix tests (bef35d4c)
- fix 2 uncatched errors (9b1aaa03)
- use the cache in priority for fast browsing at \< 20ms instead of seconds
  (81f5c3e0)
- add proper locking to avoid concurrent access (b7dc5b8a)
- many fixes and improvments at the config step (bd7b50bc)
- update install instruction with go modules (39a31040)
- test with latest nodejs and LTS (9eb271a2)
- upgrade packages + add some typescript dependencies (f105f3bb)
- transform index and App to TypeScript (aea42344)
- generate TS types for graphql queries (a2721971)
- convert bug view to TypeScript (9c570cac)
- convert bug list to typescript (6a502c14)
- convert more things to typescript (022f5103)
- convert custom tags to TypeScript (0c5f6e44)
- fix logo url (b8367082)
- typecheck remaining bug list components (e5f52401)
- force import order (9ddcb4b0)
- make travis run unit tests (76d40061)
- merge defaultRepository and repository for simplified webUI code (1effc915)
- stop using defaultRepository (465f7ca7)
- lint graphql files (b70b4ba4)
- expose the name of Repository (929480fa)
- server side take responsability to commit (0c17d248)
- fix Content type (c2d18b3a)
- finish TypeScript conversion (d0a6da28)
- run linter (c48a4dc7)
- format some files (ab09c03a)
- refactor and introduce Login and LoginPassword, salt IDs (34083de0)
- massive refactor (fe3d5c95)
- more refactor and cleanup (87b97ca4)
- pass the context to Init for when a client build process needs it (e231b6e8)
- minor fixes (a4e5035b)
- use the new generalized prompts (2792c85b)
- admittedly biased go styling (b2ca5062)
- rework to use the credential system + adapt to refactors (5c230cb8)
- fix a nil context (01b0a931)
- minor aspect fix (d7bb346d)
- create comment form (680dd91c)
- start reorganizing the component structure (8b85780d)
- move pages components (ce6f6a98)
- in the bug list, toggle open and close when clicking (d052ecf6)
- list by default only open bugs (c4f5cae4)
- fix missing space in the bug preview (602f9114)
- minor styling of the timeline events (e408ca8a)
- more styling on the bug page (86a35f18)
- fix the default query (14e91cb5)
- more readable dates, also localized (afd22acd)
- style SetStatus (218d4605)
- run linter fix (f9648439)
- fix bad formatting on Date (1164e341)
- adjust some margins (f1759ea3)
- record the login used during the configure and use it as default credential
  (0cebe1e5)
- fix label cropped in the label edition window (a322721a)
- fix bad rendering due to outdated go-runewidth (68acfa51)
- bring back the login to hold that info from bridges (purely informational)
  (893de4f5)
- correct casing for user provided login (fe38af05)
- fix tests (a90954ae)
- fix GetRemote to not break when there is no remotes (eeeb932b)
- link to other ressources (f82ad386)

## 0.6.0 (2019-12-27)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.5.0..0.6.0
```

### Bug fixes

- imported bugs count (458f4da1)
- tests (bc03a89a)
- index out of range panic in github configuration (b82ef044)
- everything following the hash-->id change (612a29b0)

### Documentation

- update implementation table (03b6afa2)
- replace images with new ones (21e82d53)
- update generated documentations (c5824ff1)
- README: make the feature-list render as list in more Markdown flavors
  (61d94305)

### Other changes

- add a apple/tomato caption (015a3b2e)
- use check marks instead of confusing fruits (3bc5e6d5)
- improve the bridge feature matrix (eb494674)
- fix ls-id description (c0c8b115)
- simplify and improve the import test (eec17050)
- polishing (239646f3)
- fix escape sequence disapearing at the end of a line (606a66dd)
- ls fix CJK characters out of alignment (5f0123d1)
- Migrate to Material-UI's new style API (d79ef7a7)
- Rework pagination (51ca8527)
- Bump dependencies (a43c7ea1)
- update dependencies (485ca590)
- add color for label (93bed322)
- format and add some comments for color label (1d94fd1b)
- rename Color to RGBColor (9839d8bc)
- use RBGA color from image/color (d156f41d)
- expose label color (9adf6de4)
- use grahql response to create labels colors (511ef010)
- Add Label gql fragment (aa6247ce)
- Implement `Authored` whenever possible (1c2ee10c)
- Render markdown (356d1b41)
- refactor how test repo are created/cleaned (c7abac38)
- add ReadConfigBool and ReadConfigString functions (d564e37b)
- add flags/config to control the automatic opening in the default browser
  (8bfc65df)
- fix Bug's Lamport edit time potentially wrong due to rebase (777ccb9c)
- generate PowerShell command completion (b64587f8)
- expose the operation when creating a new bug (08c0e18a)
- change mutations to respect the Relay specification (b2f8572c)
- consistently use `ref` to fetch a repository (9f4da4ce)
- fix typo (17cbe457)
- document the PowerShell completion (aa4464db)
- github exporter is no longer a planned feature (41a5a7fc)
- use a single KeyTarget constant for all bridges (5b1a8cde)
- detect when trying to configure a bridge with a name already taken (dc289876)
- fix a missing line break (eef73332)
- rework how RmConfigs works with git (76db2f42)
- RmConfigs usage of git version lt 2.18 (fb50d470)
- don't use the gqlgen command to generate to avoid pulling urfave/cli
  (14022953)
- fix project visibility prompt (c805142f)
- add github.com/xanzy/go-gitlab vendors (15d12fb6)
- init new bridge (01c0f644)
- init exporter (cfd56535)
- add bridge configure (a1a1d486)
- bridge project validation (35a033c0)
- add issue iterator (8ee136e9)
- remove request token methodes (51445256)
- add method to query all project labels (aea88180)
- prompt only for user provided token (6c02f095)
- fix iterator out of index bug (612264a0)
- update github.com/xanzy/go-gitlab to version 0.19.0 (1c23c736)
- fix iterator bugs and enhacements (b512108a)
- add iterator LabelEvents (89227f92)
- add import note utilities (53f99d3b)
- complete importer (8b6c8963)
- check identity cache in ensurePerson (ffb8d34e)
- check notes system field (e012b6c6)
- add snapshot.SearchComment method (d34eae18)
- make resolve error unique within the importer (76a389c9)
- add import unit tests (05a3aec1)
- fix note error handling bug (ce3a2788)
- add bridge config tests (7726bbdb)
- move constants to gitlab.go (b9a53380)
- remove exporter (5e2eb500)
- add gitlab client default timeout (b1850783)
- Fix test project path (b27647c7)
- update generated docs (54dd81e3)
- improve tests and errors (ece2cb12)
- global code and comment updates (d098a964)
- change validateProjectURL signature (0329bfdf)
- fix comment edition target hash in the import (0c8f1c3a)
- add getNewTitle tests (29fdd37c)
- handle other notes cases (5327983b)
- fix bug when running import multiple times (e678e81b)
- importer handle mentions in other issue and merge requests (ca5e40e5)
- compute op's ID based on the serialized data on disk (2e1a5e24)
- fix bad refactor (a0dfc202)
- use a dedicated type to store IDs (67a3752e)
- upgrade github/xanzy/go-gitlab version to 0.20.0 (f6280a22)
- upgrade github.com/99designs/gqlgen to v0.9.2 (d571deef)
- add context.Context to ImportAll and ExportAll signatures (5ca326af)
- use errgroup.Group instead of sync.WaitGroup (501a9310)
- silence export and import nothing events (e6931aaf)
- fix name case sensitivity in retrieving and creating labels using github
  graphql api (d19b8e1a)
- add exporter test cases for label change bug (4a4e238d)
- add exporter implementation (f1c65a9f)
- rebase and correct exporter (f1be129d)
- fix edit comment request and remove label functionalities (514dc30c)
- improve exporter error handling and label change operations (63e7b086)
- exporter ignore issues imported from or exported to different projects
  (22960159)
- remove gitlab url checking before export (c8fdaab5)
- tweaking (35c6cb6e)
- fix git version parsing with broken version (91e4a183)
- recompile the web interface (23239cc1)
- allow to cancel a cleaner (cb204411)
- also protect cancel with the mutex (c4accf55)
- minor cleanup (6a0336e0)
- add a `tui` alias for `termui` (c7792a5d)
- enhance flag description (65d7ce7c)
- add bridge configure completion scripts (77e60ace)
- recover terminal state in password prompts (be947803)
- move cleaners to where is called (46f95734)
- add tokenStdin field to bridgeParams (f3d8da10)
- update react-scripts (c56801b7)
- fix a missing key (0020e608)
- upgrade to material-ui 4 (87c64cd8)
- Fix bug listing style (0ad23d0e)
- make repository.validLabels a connection (7df17093)
- silence usage when cobra commands return an error (e5b33ca3)
- fix minor grammar issues and clarify a bit (26b0a9c9)
- reference git internals documentation (17e0c032)
- fix integration tests (8498deaa)
- iterator now query all label events when NextLabelEvent() i called, and sort
  them by ID (312bc58c)
- iterator use simple swap (ed774e4e)
- try to describe the `OperationPack` format more clearly (98792a02)
- config interface and implementation rework (ab935674)
- add ReadTimestamp methods and improve naming (7f177c47)
- add StoreTimestamp/StoreBool to the config interface (104224c9)
- use `repo.runGitCommand` and `flagLocality` instead of execFn (93048080)
- improve documentation and fix typo mistake (b85b2c57)
- update RepoCache and identity to use new repository Config (618f896f)
- use new repository configuration interface (60c6bd36)
- fix ineffectual assignment in git test (f9f82957)
- add colors for labels (d0d9ea56)
- rename RGBA to Color (75004e12)
- add labels colors in bug table (25b15169)
- fix tests (209d337b)
- better overflow management (c9e82415)
- add labels color + formatting for comments (809abf92)
- upgrade github.com/xanzy/go-gitlab dependencies to 0.21.0 (a3a431ed)
- use gitlab.Labels pointer instead of string slice (4666763d)
- support bridge imports after a given date and resumable imports (614bc5a2)
- improvement on the import resume feature (57e23c8a)
- support darwin operating systems (565ee4e4)
- improve iterator NextTimelineItem function (13f98d0c)
- add missing error check in export tests (b1a76184)
- improve iterator readability (bf84a789)
- migrate to awesome-gocui instead of the old fork I had (cb8236c9)
- rework the cursor in bugtable to match the rendering before the switch to
  awesome-gocui (965102f7)
- Implement token functionalities (a6ce5344)
- comment token functionalities (56551b6a)
- add bridge token subcommand (9370e129)
- use token id instead of name (967e1683)
- use token value as identifier (3433fa5d)
- various cleanups (3984919a)
- use a hash as token identifier instead of the token it self (baefa687)
- use entity.Id as id type (4dc7b8b0)
- store token in the global config and replace scopes with create date
  (bbbf3c6c)
- regenerate documentation and fix imports (45653bd3)
- various improvement on the global token PR (e2445edc)
- add bridge token show (f8cf3fea)
- rename `token` into `auth` (e0b15ee7)
- update github.com/xanzy/go-gitlab to v0.22.0 (83eb7abd)
- follow API changes (c1f33db2)
- fix iterator regression (e3e37fd7)
- don't forget to assign the new packs after a merge (0b2a99ab)
- esthetism rename (a9b32e6b)
- use NeedCommit() in the interface, drop HasPendingOp() (ed2ac793)
- document import/export events (67c82f4a)
- add missing metadata (8ffe2a9b)
- make sure there is no Operation's hash collision (283e9711)
- update github.com/xanzy/go-gitlab dependencies (87f86bca)
- importer corectly emit events (8b5685bb)
- export correctly emit nothing events (87244d3c)
- importer and exporter correctly emit NothingEvents (967f19a3)
- importer correctly emit NothingEvent (d6d5978b)
- update github.com/awesome-gocui/gocui dependencies (17b43299)
- sort project candidate in the interactive wizard (5054b8db)
- use the target as well in the token ID (76b61293)
- load token value in ensureInit (bf758386)
- configuration with global configs (b1d0f48f)
- use core.ConfigKeyToken instead of keyToken (014e754f)
- add bridge configure --token-id flag (09db1cda)
- add LoadTokensWithTarget and LoadOrCreateToken functions (da2d7970)
- add gitlab bridge configuration (06abb5a5)
- trim inputs during bridge configuration (7cb77209)
- tiny cleanups of the configurator (afe69d0c)
- configurator cleanup (4f856d6f)
- move export event handling to the CLI (1a1e313f)
- fix incorrect last import time on context cancel (8f7f8956)
- huge refactor to accept multiple kind of credentials (b92adfcb)
- Correctly cast configs\[configKeyKind\] (58c0e5aa)
- `user create` only assign the user identity if not set (da6591e4)
- support self-hosted GitLab instance (f6b4830c)
- allow to configure and pull without having set a user first (864d3ed3)
- add missing baseUrl prompt and options (5cffb5d1)
- fix an excessive assumption about an error (fc568209)

## 0.5.0 (2019-04-21)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.4.0..0.5.0
```

### Documentation

- update webui screenshot (2ac2c881)
- same size for the webui screenshots (d26b1d08)
- fix typos (4184beaf)
- add an architecture overview (cc3a21ac)

### Other changes

- minor cleaning (5653ae98)
- fix broken truncate with unicode and use the ellipsis character in
  LeftPadMaxLine (5e744891)
- use the '↵' symbol to save screen space (ab970da4)
- tighter column in the bug table (9c89cf5b)
- slightly better error message (a133cdff)
- simplify regex (e1714489)
- ignore jetbrains project files (85a68c82)
- Add developer-specific information. (c31e7fba)
- add more explanation about the dev process (63807382)
- minor cleaning (47b2aa4c)
- upgrade npm dependencies to fix
  https://nvd.nist.gov/vuln/detail/CVE-2018-16469 (8fc15a03)
- now that it's possible, split the schema for clarity (0d5bd6b1)
- hopefuly fix the handling of chinese (f9fc85ac)
- fix a wrapping bug leading to line longer than they should (261aa617)
- more chinese related fixes (7454b950)
- display an explicit placeholder for empty messages (94b28b68)
- build on all go and nodejs version supported (45b82de0)
- minor cleaning (96f51416)
- switch to the previous/next page when going up/down. (1174265e)
- Better position the cursor when changing page. (87098cee)
- don't reset the cursor when paginating with left/right (fb87d448)
- use a forked gocui to fix large character handling (ebcf3a75)
- fix handling of wide characters (32b3e263)
- fix non determinist zsh comp generation (3f694195)
- show: change for a single valued --field flag (43d0fe5c)
- use tiers (090cd808)
- update the date in the generated doc (09692456)
- output the build info message on stderr to avoid breaking scripts (d380b3c1)
- Add ls-id \[<prefix>\] command (f70f38c8)
- Add ls-id \[<prefix>\] command (3c0c13bb)
- fix unhandled error (0aefae6f)
- implement the loading from git (06d9c687)
- add metadata support (3df4f46c)
- more progress and fixes (bdbe9e7e)
- more progress and fixes (844616ba)
- somewhat getting closer ! (d10c7646)
- more cleaning and fixes after a code review (14b240af)
- more refactoring progress (56c6147e)
- wip push/pull (328a4e5a)
- wip (21048e78)
- add more test for serialisation and push/pull/merge + fixes (cd7ed7ff)
- I can compile again !! (d2483d83)
- all tests green o/ (da558b05)
- work on higher level now, cache, first two identity commands (864eae0d)
- fix tests (976af3a4)
- wip caching (947ea635)
- working identity cache (54f9838f)
- complete the graphql api (ffe35fec)
- store the times properly (71f9290f)
- fix tests (71930322)
- some UX cleanup (b8cadddd)
- fix RmConfigs (839b241f)
- add the clean-local-identities target for debugging (ecf857a7)
- fix typo (b59623a8)
- fix 3 edge-case failures (e100ee9f)
- simplify some code (268f6175)
- fix ResolveIdentityImmutableMetadata byt storing metadata in IdentityExcerpt
  (8bba6d14)
- add a super-fast `user ls` command (7a80d8f8)
- add a `user adopt` command to use an existing identity (304a3349)
- add a `.` at the end of Short commands usage (2fd5f71b)
- another round of cleanups (46beb4b8)
- show the last modification time in `user` (c235d89d)
- better API to access excerpts (bad05a4f)
- `user ls` also show metadata (f6eb8381)
- fix potential bug due to var aliasing (b6bed784)
- `git bug ls` should be faster (43e56692)
- make the title filter case insensitive (40865451)
- Fixing ls-id (a45ece05)
- don't make bug actions drive identity actions (a40dcc8a)
- add basic unit testing (d27e3849)
- properly push/pull identities and bugs (24d6714d)
- only return the error (not the function help) when no identity is set
  (bdf8523d)
- fix a bad output in `bug comment` (029861fa)
- display comment's id in `git bug comment` (0a71e6d2)
- Upgrade dependencies (67c84af4)
- Use Timeline API instead of raw operations (850b9db8)
- Rework timeline style (22089b5e)
- pack it (e028b895)
- expose allIdentities, identities and userIdentity in the repo (15c258cd)
- Fix and match for labels (1d758f9f)
- add a push/pull test (96987bf6)
- fix labels no showing properly in `git bug show <id> -f labels` (a64aaacc)
- add `show --field humanId` (96d356a3)
- add a --field flag to `git bug user` to display users details individually
  (5b0a92de)
- make Bug's actors and participants a connection (e027d5ee)
- fix test indentation (5733178a)
- expose valid labels (14461060)
- fix bug when trying to edit without selection (ff686e6d)
- fix ls not displaying the new Identities properly (5eeeae7c)
- fix EditCommentOperation targeting the wrong comment (d862575d)
- fix a potential crash with malformed data in EditCommentOperation (ef84fda0)
- update the documentation with the new identity workflow (5dd9d248)
- make bugTable only use the cache Easy pick (b76357a5)
- add a feature matrix of the bridges implementation (5be164c4)
- enable go 1.12, build release with go 1.11 (8d7a2c07)

## 0.4.0 (2018-10-21)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.3.0..0.4.0
```

### Bug fixes

- build (a37a5320)
- js formatting with prettier (e89375f2)

### Documentation

- update manpages due to change of month (95021a07)

### Other changes

- update gqlgen to 0.5.1 (b478cd1b)
- add a data validation process to avoid merging incorrect operations (7bec0b1f)
- better help text for the query language (bcf2b6db)
- rename 'new' into 'add' to have a verb (6b732d45)
- git bug comment now show the comments of a bug (bfb5e96a)
- add `git bug comment add` to add a comment (6cdc6c08)
- add a title command to display a bug's title (d9f72695)
- add a title edit command (ae100e0e)
- make the `commands` command show subcommands as well (b9fc8b66)
- add a `status` command to show a bug status (a846fb96)
- migrate the open/close commands under the `status` command (dad61892)
- make `label` display the current labels (cc086eba)
- add a `label add` command to add new label to a bug (2965b70f)
- add a `label rm` command to remove labels from a bug (5eaf9e83)
- add a package to handle implicit bug selection (0d5998eb)
- add a `select` command to select a bug for future implicit use (5f9fd2a2)
- convert compatible commands to the implicit select mechanism (544b9cc0)
- readBug returns better errors (84555679)
- don't ignore error when building the cache (760d0771)
- use q as keybinding to quit the show bug view (a645c901)
- explain how to quit (2daf2ddc)
- relay early the merge events (63d0b8b7)
- don't stop the process when one merge fail (4c576470)
- reclassify some merge error as `invalid` instead of hard error (1060acfd)
- fix a panic on merge invalid (d57e2fdd)
- ls now accept queries without quote (d71411f9)
- update favicon with git-bug logo (386cc3d6)
- workaround for git returning no path when inside a .git dir (8a038538)
- serve the index.html file by default to deal with the SPA router requirements
  (7c63417e)
- add the beginning of a github importer (1c86a66c)
- description cleanup (cfce3a99)
- add a `ls-labels` command that output valid labels (6e447594)
- make github 2FA work (6a575fbf)
- split the Repo interface to avoid abstraction leak in RepoCache (82eaceff)
- better interfaces, working github configurator (921cd18c)
- more documentation (c3a5213f)
- cleanup file name (a122d533)
- add functions to read/write git config (666586c5)
- big refactor and cleanup (5e8fb7ec)
- add the `bridge` and `bridge configure` commands (43bda202)
- add `bridge rm` (061e83d4)
- add `bridge pull` (2282cbb5)
- validate config before use (c86e7231)
- query most of the data (c4a20762)
- add the ability to store arbitrary metadata on an operation (a72ea453)
- add the optional field AvatarUrl to Person (5d7c3a76)
- add raw edit functions to allow setting up the author, the timestamp and the
  metadatas (40c6e64e)
- add a target producing a debugger friendly build (25bec8eb)
- first working github importer (879e147e)
- add a general test for the handler/resolvers (f9693709)
- detect when the title is not changed and abort the operation (ac29b825)
- detect when an edit title doesn't change it and abort the operation (18f5c163)
- add a `deselect` command to deselect a previously selected bug (04ddeef9)
- don't forget to treat the error when selecting a bug (86792d78)
- clear the selected bug when invalid (66f3b37c)
- better responsive columns in the bug table (5b3a8f01)
- handle both sha1 and sha256 git hashes (8ab2f173)
- manually fix the generated code, gix the graphql handler (8af6f7d9)
- fix a link (8fdd6bf9)
- define a hash-based identifier for an operation (794d014f)
- apply an operation with a pointer to the snapshot (41e61a67)
- implement comment edition (c46d01f8)
- expose the new Timeline (36ebbe0c)
- fix compilation (75c921cd)
- various minor improvements (037f5bf5)
- advertise edited comments (bad9cda9)
- use deditated type for all TimelineItem (7f86898e)
- use a value embedding for OpBase (3402230a)
- add a test for OpBase metadata (bda9b01b)
- add a test for operations hash (97d94948)
- `bridge` don't take arguments (a4be82ca)
- also index the first op metadata (be59fe0d)
- add a new no-op operation to store arbitrary metadata on a bug (de81ed49)
- also clear the cache after deleting the bugs (aea85f04)
- custom error for the different error case when loading a bug (f026f61a)
- in op convenience function, return the new op to be able to set metadata later
  (6ea6f361)
- message can be empty on edit comment (0fe7958a)
- make sure to invalidate the hash when changing an op's metadata (f18c2d27)
- working incremental + comment history for the first comment (8ec1dd09)
- incremental import of comments + editions (892c25aa)
- incremental import for labels, title edition, status changes (b5025a51)
- better multi choice prompt to select the target for `bridge configure`
  (f37155d0)
- explain better what happen with the user credentials (f4643632)
- handle the case where no diff is available for a comment edition (558e149b)
- deal with the deleted user case where github return a null actor (64133ee5)
- add missing operation (03202fed)
- also pull users email (7cb7994c)
- update packed files (e414a0e3)
- some cleanup in the label edition code (7275280d)
- don't load the repo for commands that don't need it (7a511f9a)
- fix `comment add` flags set on the wrong command (b08e28e6)
- check the bug id before the user write the message for `comment add`
  (f67c57c0)
- unify the processing from editor/file/stdin for `add` and `comment add`
  (d37ffa6b)
- add a new SetMetadataOperation to retroactively tag operations (82701f8c)

## 0.3.0 (2018-09-13)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.2.0..0.3.0
```

### Bug fixes

- english grammar (30d4bc21)

### Documentation

- fix some typos (73bd0f4a)
- add logo to README.md (e3265303)
- add missing period in README.md (33a0ae2b)
- fix terminal UI recording src (2078100e)
- document the query DSL (28ee08af)

### Other changes

- fix a crash when a bug is created with an empty message (c974cc02)
- add the gitter badge (6ecfb9da)
- advertise a little more the chat lobby (71523f23)
- remove use of the too recent %(refname:lstrip=-1) of git (b5881213)
- add benchmarcks for bug merge (08127d8d)
- make it seedable and reusable (285e8394)
- add a benchmark for reading all bugs in a repo (8575abf2)
- Format everything with prettier (bb4ebed0)
- Ensure code format in CI by running eslint (ce2be02c)
- added archlinux aur package in install section (4a2fedd9)
- a bit of styling (fd268767)
- more styling (94217828)
- lock the repo with a pid file; automatic cleaning (6d7dc465)
- introduce WithSnapshot to maintain incrementally and effitiently a snapshot
  (16f55e3f)
- add a new BugExerpt that hold a subset of a bug state for efficient sorting
  and retrieval (e7648996)
- maintain, write and load from disk bug excerpts (0514edad)
- add name to web app manifest. (11ad7776)
- rename RootCache into MultiRepoCache (90a45b4c)
- provide sorted (id, creation, edit) list of bugs (919f98ef)
- update (e3c445fa)
- provide a generic bug sorting function (0728c005)
- make sure the lamport values are set properly after a commit (e2a0d178)
- fix missed code path that should update the cache (c0d3b4b0)
- add proper licensing and small cleaning (e82b92f6)
- add logotype as well (74c48ca0)
- also update the operations incrementaly in the snapshot (d17cd003)
- fix the logo url to use to master branch (56333087)
- recomend go get -u (453ae857)
- only print once the error (6f1767d1)
- various cleaning (f136bf6a)
- clean outdated build tag (265ecd81)
- refactor the Pull code to have the message formating in the upper layers
  (61a1173e)
- refactor to handle bug changes during Pull (6d7e79a2)
- add a function to parse a status (877f3bc2)
- add a function to test the matching of a query (13797c3b)
- implement the filtering (a38c1c23)
- also store bug labels (21f9840e)
- combine sorting and filtering into a query with its micro-DSL (09e097e1)
- accept a query to sort and filter the list (dd0823dd)
- add an example of query (71bee1e6)
- properly parse and clean qualifier with multi word (0dc70533)
- add the alias `state` for the qualifier `status` (ece9e394)
- doc & cleaning (c8239a99)
- support expressing a query with flags as well (9bb980e9)
- ensure that OpBase field are public and properly serialized (2dcd06d1)
- resolved id by prefix using the cache instead of reading bugs (d1c5015e)
- use Esc key to quit instead of 'q' to free it for a `query` feature (30e38aab)
- allow to change the bug query (9cbd5b4e)
- AllBugs now accept a query (7b05983c)
- change the OperationPack serialization format for Json (60fcfcdc)
- proper int baked enum for merge result status instead of a string (19f43a83)
- add missing query help text (8a25c63d)
- return a more convenient array of result for label changes (f569e6aa)
- better perf by ensuring that the folder is created only once (bf11c08f)
- use 'q' for quit and 's' for search (f8b0b4f5)
- attempt to future-proof the cache file (b168d71f)

## 0.2.0 (2018-08-17)

To view the full set of changes, including internal developer-centric changes,
run the following command:

```
git log --oneline 0.1.0..0.2.0
```

### Bug fixes

- some linting trouble (df144e72)
- tests (1e9f2a9d)

### Other changes

- revamp the bug list (5edcb6c8)
- don't pack the huge .map file for production (43f808a0)
- expose startCursor and endCursor as well for a connection (ef0d8fa1)
- add a small program to go:generate the code (5c568a36)
- fix two bugs in the connection code (bc1fb34c)
- implement pagination on the bug list (24d862a6)
- reorganize the code (2530cee1)
- rework of the bug page with a timeline (1984d434)
- display label changes in the timeline + cleaning evrywhere (cf9e83e7)
- add `was` on SetTitleOperation to store what the title was (a4740937)
- display title changes in the timeline (17aa4050)
- display status change in the timeline (11b79260)
- pack it (f728a02a)
- minor css improvements (51b0d709)
- add a target to remove all local bugs (f510e434)
- fix out of bounds when opening a bug on non-first page (6af16c1c)
- show the bug after creation (e482a377)
- add a target to clean bugs on a remote (90f235b3)
- fix left/right unnecessarely moving up/down (c93c0221)
- update with new recording of the termui, and screen of the webui (55ab9631)
- fix the termui screencast not working on github (4e9ff2f5)
- cleanup (1e8e1af6)
- pack it (e076931a)
- create less bugs (eaef3149)
