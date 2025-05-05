#!/usr/bin/env bash

# use //:.gitmessage as the commit message template
git config --local commit.template ".gitmessage"

# use a common, shared file as the default for running git-blame with the
# `--ignore-revs` flag
git config --local blame.ignoreRevsFile ".git-blame-ignore-revs"

# enable features.manyFiles, which improves repository performance by setting
# new values for several configuration options:
#   - `core.untrackedCache = true` enables the untracked cache
#   - `index.version = 4` enables path-prefix compression in the index
#   - `index.skipHash = true` speeds up index writes by not computing a trailing
#     checksum
git config --local features.manyFiles true
