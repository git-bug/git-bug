# ADR 0004: External auth implementation phasing

## Status
Accepted

## Context

Adding external authentication (OAuth/OIDC via GitHub, Google, and eventually atproto/Bluesky) to the public webui involves several interlocking concerns: the OAuth flow itself, session management, identity import and adoption, public key management, operation commit signing, and a new ProjectConfig entity for role management.

These concerns are not equally urgent. The OAuth flow and session management are required to ship a usable public webui. The key management, signed web operations, and ProjectConfig are security hardening and can ship incrementally without blocking the initial release.

A single-phase implementation risks either delaying the webui indefinitely while key infrastructure is built, or shipping a half-implemented key model that is hard to evolve.

## Decision

External auth is implemented in five independent phases. Each phase ships working, non-broken software. Later phases harden the security model without requiring changes to earlier deployments.

### Phase 1 — External auth (MVP)
OAuth/OIDC via `goth` (GitHub and Google). Session JWT signed with an in-memory auto-generated secret (override via `--auth-secret` / `GIT_BUG_AUTH_SECRET`). On first login: create or find Identity via `ResolveIdentityImmutableMetadata`, store provider's numeric user ID as Immutable Metadata. Operations attributed to the Identity and **unsigned**. The Identity's `keys` field is left empty — no server keypair, no imported provider keys in the signing field. This keeps Identities unprotected and avoids any signing requirement.

Provider credentials (client ID + secret) are supplied via flags or environment variables on `git bug webui`. No interactive configuration ceremony.

### Phase 2 — Key import
Change `Key` storage format to `publicKeyMultibase` (ADR-0001) and update signing to support SSH and PGP formats (ADR-0002). Import public keys from providers at identity creation time: GitHub GPG keys API for GitHub users; `user.signingkey` + `gpg.format` from local git config at `git bug user new`. Imported keys populate the `keys` field, making Identities protectable. Key generation opt-in: `git bug user key generate`.

### Phase 3 — Signed web operations
Implement ServerIdentity and AdminRole (ADR-0003). `git bug webui setup` creates the ServerIdentity, declares it admin in local git config (transitional until Phase 4). Web users with unprotected Identities now get cryptographically signed, verifiable operation commits. Users with protected Identities are unaffected.

### Phase 4 — ProjectConfig
Implement the ProjectConfig entity (distributed, versioned, DAG-based). Move AdminIdentity declarations from local git config into ProjectConfig. Roles become verifiable by any client offline. ProjectConfig also absorbs other currently-local config: allowed labels, bridge configuration, webui settings.

### Phase 5 — atproto / Bluesky
atproto OAuth via a custom `goth` Provider (PKCE + DPoP). DID resolution and signature verification via `go-did-it`. PLC key rotation tracked by re-fetching the DID document on each login and on signature verification failure. Identity versioning handles key rotation naturally via `ValidKeysAtTime`.

## Consequences

- Phase 1 ships a working public webui with no key infrastructure. Operations are unsigned but correctly attributed via OAuth session.
- Unsigned operations from web users remain valid in all future phases (backward compatible). Phase 3 adds signed operations as an improvement, not a breaking change.
- The `keys` field being empty in Phase 1 is intentional: populating it without a corresponding server signing key would make web-created operations fail verification. Phase 2 and 3 are designed to be adopted together or in sequence.
- Each phase is independently deployable. Operators who never need signed web operations can stay at Phase 1 indefinitely.
- The phasing defers the most complex decisions (ProjectConfig schema, atproto DPoP implementation) until earlier phases have validated the overall approach.
