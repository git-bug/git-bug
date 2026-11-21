# Security policy

## Reporting a vulnerability

Report it privately, through [GitHub's private vulnerability reporting][report]. Please don't open a public issue, pull request or discussion.

A useful report says:

- which version or commit is affected, and how git-bug is used (CLI, termui, webui, bridge)
- what an attacker needs: push access, access to a webui, a crafted repository, a visit to a malicious website…
- what they get, and the steps to reproduce it

git-bug is maintained by volunteers. We aim to acknowledge a report within a week, and to keep you informed until it is resolved.

[report]: https://github.com/git-bug/git-bug/security/advisories/new

## Supported versions

git-bug is not yet at 1.0. Fixes go into `trunk` and the next release; older releases are not patched at the moment.

## Scope

The [threat model](doc/threat-model.md) describes who git-bug trusts, what it guarantees, and which threats it accepts. Read it before reporting: it decides most of what is in scope.

In scope:

- breaking convergence or integrity, as the threat model defines them
- getting past the read-only mode of the webui
- attacks on a webui through a website its user visits (request forgery, cross-origin access), or through the content it renders, including content imported by a bridge
- running code, or reading or writing files, through a crafted repository, config or bridge response
- crashing or hanging git-bug through crafted data, beyond the denial of service the threat model accepts

Out of scope:

- what the threat model accepts, doesn't guarantee, or lists as not done yet, such as a pusher on a plain git host with direct git access
- a writable webui reachable beyond a trusted network: whoever can reach it acts as its user
- a vulnerability in a dependency with no demonstrated impact on git-bug: those are tracked by automated updates

When unsure, report it: we would rather close a report than miss one.

## Disclosure

Once a fix is released, we publish a GitHub security advisory, request a CVE when warranted, and credit you unless you prefer otherwise. Please keep the issue private until then.
