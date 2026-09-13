# Design: External Authentication and Identity Key Management

This document is the master plan for adding external authentication to the public webui and evolving git-bug's identity key model. It covers all five implementation phases, their internal design, dependencies, and open questions.

Related artefacts:
- `CONTEXT.md` — domain glossary; all terms used here are defined there
- `docs/adr/0001` — public key storage format (publicKeyMultibase)
- `docs/adr/0002` — operation commit signing (SSH / PGP format)
- `docs/adr/0003` — ServerIdentity and AdminRole impersonation
- `docs/adr/0004` — phasing rationale and phase boundaries
- `docs/prd-external-auth.md` — Phase 1 PRD with user stories and test decisions

---

## Current State

The webui runs in two modes today:

- **Read-only**: no authentication, all write operations rejected.
- **Single-user**: a fixed `entity.Id` (the local git config user) is injected into every request context by `api/auth.Middleware`. There is no login flow; the server acts as the repo owner permanently.

The `api/auth` package has three files: `middleware.go` (injects a fixed ID), `context.go` (stores/retrieves the ID from context), `errors.go`. The GraphQL resolvers call `auth.UserFromCtx` to obtain the `*cache.IdentityCache` for the current request.

Identities have a `keys []*Key` field per version. The `Key` type wraps `*packet.PublicKey` from OpenPGP. The `user new` command creates Identities with `nil` keys. No UX exists to generate or attach keys. `IsProtected()` is always false in practice. The signing path in `entity/dag/operation_pack.go` silently skips signing when `SigningKey()` returns nil.

The GitHub bridge imports Identities with `nil` keys and stores only the GitHub login as immutable metadata — not the numeric user ID, and not any public keys from GitHub's API. This is a known gap that Phase 2 addresses.

---

## Phase 1 — External Auth MVP

**Goal:** A public webui where external users can log in via GitHub or Google, be mapped to a git-bug Identity, and create operations attributed to that Identity.

**Ships:** OAuth flow, session management, identity import/lookup. No key management.

### OAuth / OIDC library

Use `markbates/goth`. It provides a unified `Provider` / `Session` interface across GitHub (OAuth 2.0, non-OIDC) and Google (OIDC). Its interface is extensible — a custom `Provider` implementation is the intended path for atproto in Phase 5.

### Provider configuration

No interactive setup. Credentials are supplied via flags on `git bug webui` or environment variables:

```
--github-client-id        GIT_BUG_GITHUB_CLIENT_ID
--github-client-secret    GIT_BUG_GITHUB_CLIENT_SECRET
--google-client-id        GIT_BUG_GOOGLE_CLIENT_ID
--google-client-secret    GIT_BUG_GOOGLE_CLIENT_SECRET
--auth-secret             GIT_BUG_AUTH_SECRET        (JWT signing secret, see below)
```

If no provider credentials are present, the server starts in single-user mode exactly as today. OAuth routes are not registered.

Each provider requires a redirect URI registered with the provider's developer console pointing to `<base-url>/auth/<provider>/callback`. The operator registers their own OAuth app — git-bug does not ship a shared app for the webui (unlike the bridge, which uses a git-bug org GitHub app with the device flow).

### Session management

After a successful OAuth callback, the server issues a **server-signed JWT** stored as an HTTP-only cookie. The JWT payload:

```
{ "sub": "<entity.Id>", "exp": <unix timestamp> }
```

The signing secret (`GIT_BUG_AUTH_SECRET`) is:
- Auto-generated randomly in memory if not set. All sessions are lost on server restart — this is acceptable for the common single-instance self-hosted case.
- Set explicitly for persistence across restarts or load-balanced deployments.

The `api/auth.Middleware` function gains a new variant that extracts and validates the JWT cookie and calls `CtxWithUser`. The existing fixed-user variant is retained for single-user mode.

Session expiry: configurable, defaulting to 24 hours. Refresh is not implemented in Phase 1 — users re-authenticate on expiry.

### HTTP routes

New routes registered on the gorilla/mux router:

```
GET  /auth/{provider}           — initiate OAuth redirect via goth
GET  /auth/{provider}/callback  — OAuth callback: validate, import identity, set cookie, redirect
POST /auth/logout               — clear session cookie
GET  /auth/user                 — return current user info (for webui header display)
```

### Identity import and lookup

On every successful OAuth callback, the import logic runs:

1. Extract `goth.User.UserID` — the provider's stable numeric/opaque user ID (not the username).
2. Call `repo.Identities().ResolveIdentityImmutableMetadata("<provider>:user-id", userID)`.
3. **Found**: use the existing Identity. Done.
4. **Not found**: attempt adoption — if a default user Identity is set (single-user mode), offer to tag it with the External Account Reference metadata (follows `bridge/core.FinishConfig` pattern). The operator can pre-run `git bug user link --provider <provider>` locally to set this up before going public.
5. **Still not found**: create a new Identity via `repo.Identities().NewRaw(name, email, login, avatarURL, nil, map[string]string{"<provider>:user-id": userID})`. The `keys` field is nil — intentional, see below.

The External Account Reference key format is `"<provider>:user-id"` where provider is `github`, `google`, etc. The value is always the provider's numeric/opaque user ID (e.g. `"12345678"` for GitHub), never the mutable username.

### Why `keys` is nil in Phase 1

If keys were imported into the `keys` field, the Identity would become protectable. Once any signed commit exists in the chain, `IsProtected()` returns true and all subsequent operation commits for that Identity must be signed. The server does not hold private keys in Phase 1, so it cannot sign — and unsigned commits from a protected Identity fail verification. Leaving `keys` nil avoids this entirely.

Imported provider public keys (GitHub GPG keys, SSH keys) are stored in Phase 2. Until then, web-created operations are unsigned, which is acceptable — unsigned commits from unprotected Identities are valid, and the OAuth session is the trust anchor.

### Existing mode non-regression

Single-user mode: `git bug webui` with no provider credentials → fixed-user middleware, no OAuth routes, identical behaviour to today.

---

## Phase 2 — Key Format and Import

**Goal:** Replace the broken OpenPGP key serialisation with a format that is algorithm-agnostic, DID-compatible, and works with modern keys. Import user public keys from git config and provider APIs.

**Depends on:** Nothing from Phase 1 (can ship in parallel or before). Is a prerequisite for Phase 3.

### Key storage format change (ADR-0001)

The `Key` type is refactored. Public keys are stored as `publicKeyMultibase`: a base58btc-encoded string with a multicodec varint algorithm prefix and raw key bytes. This is the W3C DID `publicKeyMultibase` format, implemented by `go-did-it`.

The `identity/version.go` `formatVersion` is bumped from 2 to 3. Migration: no migration is needed because no deployed Identity currently has keys attached.

The `Key` struct:

```go
type Key struct {
    publicKeyMultibase string  // always present
    pgpEntity          string  // omitempty: full armored GPG entity, GPG-origin keys only
    // private key is NOT stored here; held in OS keyring or agent
}
```

For **GPG-origin keys**: parse the full armored GPG entity to extract raw public key bytes → encode as multibase. Store both. The entity is authoritative; multibase is derived from it. The entity carries the User ID and self-signed certification needed for `gpg --import`.

For **SSH-origin keys** and **generated Ed25519 keys**: `publicKeyMultibase` only. The SSH authorized_keys wire format is trivially reconstructable from multibase + algorithm. No extra field needed.

The `PGPEntity()` method on `Key` is removed. The OpenPGP dependency is retained only for: (a) signing GPG-origin operation commits, (b) parsing `pgpEntity` blobs at import time.

### Key import at identity creation

`git bug user new` and `git bug user adopt` read `gpg.format` and `user.signingkey` from git config (via `repo.AnyConfig()`) and import the key:

- `gpg.format = ssh` and `user.signingkey = /path/to/key.pub`: read the SSH public key file, encode as multibase, populate `keys`.
- `gpg.format = openpgp` and `user.signingkey = <fingerprint>`: fetch the full GPG entity from the local GPG keyring (`gpg --export <fingerprint>`), store as `pgpEntity` + derived `publicKeyMultibase`.
- Neither set: display an informational message (see below). Do not generate a key.

The informational message when no signing key is found:

> No signing key found (`user.signingkey` is not set in git config).  
> Operations you create will not be cryptographically signed.  
> To enable signing: configure `user.signingkey` and re-run `git bug user adopt`,  
> or run `git bug user key generate` to create a new Ed25519 key.

### Key import from providers

At OAuth identity creation (Phase 1 creates Identity with nil keys; Phase 2 imports keys instead):

- **GitHub**: call `GET /users/{login}/gpg_keys` — returns armored GPG entities. Parse each, store as `pgpEntity` + derived `publicKeyMultibase`. Call `GET /users/{login}/keys` — returns SSH public keys. Parse each, store as `publicKeyMultibase`. Multiple keys are supported (one `Key` per entry).
- **Google**: no provider key API; skip.
- Identities with imported keys are now potentially protectable, but remain unprotected until a *signed* Identity version commit exists. The server does not sign Identity commits in Phase 2 (that is Phase 3). Importing public-only keys does not call `SigningKey()`, which requires a private key; therefore no signed commit is produced; `IsProtected()` remains false.

### Key generation (explicit opt-in)

```
git bug user key generate
```

Generates an Ed25519 keypair. Stores the private key in the OS keyring via `99designs/keyring` (already used by bridge credentials). Stores `publicKeyMultibase` in a new Identity version. Optionally offers to configure `user.signingkey` in git config to use this key for regular git commit signing.

This is the only place git-bug stores a private key directly. For keys imported from git config, git-bug delegates signing to the SSH agent (`golang.org/x/crypto/ssh/agent`) or GPG agent (`gpg --sign`), not the keyring.

### Operation commit signing update (ADR-0002)

The `StoreSignedCommit` signature in `repository/repo.go` is changed from accepting `*openpgp.Entity` to accepting a signer interface. Two implementations:

- **SSH signer**: uses `golang.org/x/crypto/ssh` (already a transitive dependency). Produces `-----BEGIN SSH SIGNATURE-----` format (`PROTOCOL.sshsig`) for Ed25519 and SSH-RSA keys.
- **PGP signer**: uses `github.com/ProtonMail/go-crypto/openpgp`. Produces `-----BEGIN PGP SIGNATURE-----` for GPG-origin keys.

Verification in `entity/dag/operation_pack.go`: a format-detecting dispatcher replaces `openpgp.CheckDetachedSignature`. It inspects the PEM header of the stored signature and routes to the appropriate verify function.

The `deArmorSignature` helper in `repository/common.go` is replaced or extended to handle both formats.

---

## Phase 3 — Signed Web Operations

**Goal:** Operations created via the webui are cryptographically signed and independently verifiable by any client, not just trusted via OAuth session.

**Depends on:** Phase 1 (auth infrastructure), Phase 2 (key infrastructure).

### ServerIdentity

The server is represented as a real git-bug Identity, created during `git bug webui setup` and stored in the repo under `refs/identities/<server-id>`. It is a regular Identity and uses all existing Identity machinery:

- **Multi-key**: each server instance in a load-balanced deployment holds a separate key in the ServerIdentity version. All instances can sign.
- **Key rotation**: create a new Identity version signed by the old key. The audit trail lives in the git object chain.
- **Revocation**: remove a key from a new Identity version.

`git bug webui setup` generates the ServerIdentity, writes the server's private key to the keyring, and declares the Identity as AdminIdentity in the repo's local git config (transitional until Phase 4).

### AdminRole and impersonation (ADR-0003)

Verification in `operation_pack.go` gains one new rule:

```
author has keys       → commit must be signed by one of author's valid keys (unchanged)
author has no keys    → if commit is signed by a declared AdminIdentity: accepted
author has no keys    → if commit is unsigned: accepted (no cryptographic guarantee)
```

AdminRole applies **only to unprotected Identities**. An Identity with keys retains exclusive signing authority. The server cannot impersonate a protected Identity.

On each operation commit for a web user with an unprotected Identity, the server:
1. Looks up the user's Identity in cache.
2. Checks `IsProtected()` — false (unprotected).
3. Signs the commit with the ServerIdentity's current key (`SigningKey()` on the ServerIdentity).
4. The commit is attributed to the web user but signed by the ServerIdentity — the standard admin impersonation path.

### Protected Identity + webui (adoption with server key)

A user with a locally-protected Identity (has their own key K1) who wants webui access must explicitly authorize the server. The flow:

1. User logs into webui via OAuth — server finds Identity via External Account Reference metadata.
2. Identity is protected. Server cannot impersonate.
3. Webui displays the ServerIdentity's current public key fingerprint.
4. User runs locally: `git bug user key add <server-pubkey-multibase>` — creates a new Identity version `[K1, KS_pub]`, signed by K1, and pushes.
5. Server now has KS_pub in the user's Identity. When the user creates web operations, the server signs with KS_priv (held in its keyring per-user). Commits are signed by a key valid for this Identity → verified by any client.

The user can revoke server access at any time by creating a new Identity version without KS_pub, signed by K1 — fully offline, no server involvement.

### Key adoption for OAuth-imported Identities

For Identities created by OAuth (no local key, server has KS_priv from Phase 1 onward... wait, Phase 3 — server has KS):

A user who authenticates via OAuth and then wants to use the same Identity locally (CLI):

1. The server holds KS_priv for this Identity. Identity has keys `[KS_pub]` → protected.
2. User runs `git bug user adopt --server <url> --provider github` locally.
3. Local git-bug generates K1, authenticates with GitHub OAuth → server verifies.
4. Server signs a new Identity version `[KS_pub, K1_pub]` with KS_priv.
5. User pulls → has K1_priv locally, Identity has both keys.
6. User can then optionally remove KS_pub: `git bug user key remove <KS_pub>`, signed by K1_priv → server loses write access.

### Non-protected Identity adoption (Phase 1 path)

If the Identity has no keys (Phase 1 import, no provider keys available), adoption is:

1. User runs `git bug user adopt --provider github` locally — authenticates with OAuth.
2. Server confirms the match (OAuth token matches the External Account Reference in the Identity).
3. No cryptographic proof is stored — the Identity is unprotected and there is nothing to prove. The user simply pulls the repo and gains local access to their Identity.
4. This is correct: unprotected means no key chain to verify against.

---

## Phase 4 — ProjectConfig

**Goal:** Distribute project-wide configuration (roles, labels, webui settings) via the git object chain so any client can verify it offline without trusting local config files.

**Depends on:** Phase 3 (AdminRole must exist before it can be stored in ProjectConfig).

### Entity model

ProjectConfig is a new distributed entity type using the DAG operation framework (like bugs). Stored under `refs/git-bug/config`. Pushed and pulled with the repo.

Operations include: `DeclareAdminOp`, `RevokeAdminOp`, `SetLabelOp`, `SetWebuiSettingOp`, and others. Each operation is appended to the DAG and signed by an AdminIdentity key.

Bootstrap: the first operation (`DeclareAdminOp` for the repo owner / ServerIdentity) is unsigned. This is the same pattern as Identity's first version. Subsequent operations require an existing AdminIdentity signature.

### Migration from local git config

The transitional local git config entry for AdminIdentity (set in Phase 3) is migrated to ProjectConfig during `git bug webui setup` (or a migration command). After migration, local config is no longer authoritative.

### Contents

- AdminIdentity declarations: list of Identity IDs with AdminRole
- Allowed labels: the set of valid label values (currently unconstrained)
- Webui settings: read-only mode flag, enabled auth providers
- Bridge configuration: currently local-only in git config; moving it here makes it shareable across team members (no more per-machine bridge reconfiguration)

### Merge strategy

Operations in the DAG are concurrent-safe for non-conflicting keys (last-write-wins per key, ordered by Lamport time). Conflicting admin operations (e.g. two admins simultaneously revoking each other) are resolved by the Lamport clock ordering.

---

## Phase 5 — atproto / Bluesky

**Goal:** Allow Bluesky users to authenticate with the webui and be attributed correct Identities including their DID-based public keys.

**Depends on:** Phase 1 (auth infrastructure), Phase 2 (publicKeyMultibase for DID key storage).

### atproto OAuth

atproto uses OAuth 2.0 + PKCE + DPoP (Demonstration of Proof-of-Possession). Key differences from standard OIDC:

- **No pre-registered client secret.** atproto uses public clients. The client ID is a URL pointing to a metadata document served by the git-bug instance (e.g. `/.well-known/oauth-client-metadata`).
- **DPoP**: each token request and API call is bound to an ephemeral client keypair. The client generates a fresh Ed25519 or P-256 keypair per session, signs requests with it, and includes the public key in a DPoP header.
- **Authorization server discovery**: the user's PDS endpoint is discovered by resolving their handle to a DID, then reading the `#atproto-pds` service endpoint from the DID document.

Implementation: a custom `goth.Provider` implementation for atproto. The `Provider` and `Session` interfaces are the natural extension point (as chosen in Phase 1). The DPoP signing uses `golang.org/x/crypto` which is already a dependency.

The `/.well-known/oauth-client-metadata` endpoint must be served by the webui at a stable public URL. This is a prerequisite for atproto auth — the server must be publicly reachable.

### DID resolution and key import

After successful atproto OAuth, the user's DID is known (e.g. `did:plc:abcdef1234567890abcdef12`). Key import:

1. Resolve the DID document via `go-did-it` (`did:plc` verifier, fetches from `plc.directory/<did>/data`).
2. Extract `verificationMethods` from the document — these are the user's current signing keys, already in `publicKeyMultibase` format.
3. Store them in the Identity's `keys` field. No conversion needed; the format is identical.

The External Account Reference: `"atproto:did": "did:plc:abcdef..."`. The DID is the stable identifier — unlike a Bluesky handle, a DID never changes.

### PLC key rotation

The PLC directory publishes key rotations at `plc.directory/<did>/log`. git-bug tracks rotations without polling:

- **On every atproto login**: re-fetch the DID document. If keys differ from the current Identity version, create a new Identity version with the updated keys. Unsigned (the old PLC key's private side is on the PDS, not locally accessible) — the Identity remains unprotected from git-bug's perspective (public keys only, no private key in keyring).
- **On signature verification failure**: lazy re-fetch as a fallback. If the DID document has newer keys, the failure may be due to a rotation that happened between the user's last login and the current verify attempt. Retry verification with the new keys.

`ValidKeysAtTime` handles the history correctly: the new Identity version's Lamport time marks the boundary. Operations before that time are verified against the old keys; operations after are verified against the new keys.

### Private key access for atproto users

Most bsky.social users do not have local access to their PDS signing key. Their Identity has imported public keys but no private key in the keyring. Web operations via the webui are signed by the ServerIdentity (AdminRole path, since the Identity is unprotected — public-only keys do not trigger `IsProtected()`). Locally, the user cannot sign operations.

Users with a self-hosted PDS who do control their signing key can run `git bug user key import --from-atproto` (future tooling) to store the key locally and gain local signing capability.

---

## Cross-Cutting Concerns

### Private key storage

git-bug accesses private keys via three paths, in priority order:

1. **OS keyring** (`99designs/keyring`, already used for bridge credentials): for keys generated by `git bug user key generate`. Cross-platform but may be unavailable in headless/container environments.
2. **SSH agent** (`golang.org/x/crypto/ssh/agent`): for SSH-origin keys. The agent is contacted at `$SSH_AUTH_SOCK`. Works in most developer environments and CI.
3. **GPG agent** (shell out to `gpg --sign`): for GPG-origin keys. Used only when `gpg.format = openpgp`.

Private keys are never stored in git objects.

### Key management UX principles

- **No keys by default.** `git bug user new` does not generate a key. It shows an informational message if `user.signingkey` is not configured.
- **Import from existing setup.** The path of least friction: if you already sign git commits, `git bug user new` imports your key automatically.
- **Explicit opt-in for generated keys.** `git bug user key generate` is a deliberate action. It stores a private key in the OS keyring and optionally configures `user.signingkey` in git config.
- **No keys forced on solo users.** A single developer using git-bug locally has no threat model requiring signed operations. Keys become meaningful when collaborating on a public instance.

### `IsProtected()` semantics

An Identity is protected once any signed commit exists in its chain. Once protected, all subsequent Identity version commits must be signed by a valid current key. Operation commits attributed to a protected Identity must be signed by a valid key (or the AdminRole exception applies — but AdminRole does not apply to protected Identities).

Importing public-only keys does not make an Identity protected, because `SigningKey()` returns nil (no private key in keyring/agent) and the Identity version commit is stored unsigned.

### Bridge gap (independent fix)

The GitHub bridge currently imports Identities with `nil` keys and stores the GitHub *login* (username) as metadata, not the numeric user ID. This should be fixed independently:

- Change `metaKeyGithubLogin` lookup to also store `metaKeyGithubId` (numeric user ID) as the canonical External Account Reference.
- In `ensurePerson`, fetch GPG keys from `GET /users/{login}/gpg_keys` and populate `keys`.

This fix is compatible with Phase 1 (bridge imports and OAuth imports will share the same External Account Reference key format) and is a prerequisite for correct identity unification between bridge-imported and OAuth-imported Identities representing the same person.

---

## Dependency Graph

```
Phase 1 (OAuth/session/identity import)
    │
    ├── Phase 2 (key format + import)  ←── independent, can ship before Phase 1
    │       │
    │       └── Phase 3 (ServerIdentity + AdminRole)
    │                   │
    │                   └── Phase 4 (ProjectConfig)
    │
    └── Phase 5 (atproto)  ←── requires Phase 1 + Phase 2
```

Phase 2 has no runtime dependency on Phase 1. It is a data model change and can ship as a standalone release. It must ship before Phase 3.

Phase 5 can be developed in parallel with Phases 3 and 4 once Phases 1 and 2 are complete.

---

## Open Questions

**Q1: Bridge External Account Reference migration.**
The GitHub bridge currently uses the login (username) as the metadata key for identity lookup (`metaKeyGithubLogin`). This conflicts with the `"github:user-id"` format established for OAuth login. If someone has bridge-imported Identities and later logs in via OAuth, they will get duplicate Identities. A migration or dual-lookup is needed. Decision deferred.

**Q2: Session refresh.**
Phase 1 uses 24-hour JWTs with no refresh. Users must re-authenticate on expiry. Is this acceptable, or should a sliding-window refresh be added in Phase 1 or Phase 2?

**Q3: Multi-provider Identity linking.**
If the same human authenticates via GitHub and then via Google, they get two separate git-bug Identities. Should there be a "link accounts" flow that merges them into one Identity? If so, which is the primary? This is not addressed in any phase currently.

**Q4: ProjectConfig bootstrap authority.**
The first `DeclareAdminOp` in ProjectConfig is unsigned (no admin exists yet to sign it). Any client can verify the chain from that point forward, but cannot cryptographically verify that the bootstrap admin is legitimate — they have to trust whoever seeded the repo. Is this acceptable, or is there a stronger bootstrap mechanism worth designing?

**Q5: ServerIdentity key per user vs. single key.**
Phase 3 describes the ServerIdentity signing operations attributed to web users. The server holds the ServerIdentity's private key. For protected Identity users who add KS_pub to their Identity, the server signs with the same KS for all such users, or generates a per-user key. The current design implies the ServerIdentity has one (or a few, for multi-instance) keys shared across all users it signs for. Is this the right model, or should there be per-user keys?

**Q6: atproto without public server URL.**
atproto OAuth requires the server to serve `/.well-known/oauth-client-metadata` at a public URL. If the server is behind a NAT or running locally, atproto OAuth is unavailable. This should be documented clearly. Other providers (GitHub, Google) work fine locally via redirect URIs.

**Q7: Key revocation propagation.**
If a user revokes the server key (removes KS_pub from their Identity), the server needs to notice this on the next request and refuse to sign on their behalf. The server must re-read the Identity's key set on each request (or on session validation), not cache it indefinitely. Caching strategy is not specified.
