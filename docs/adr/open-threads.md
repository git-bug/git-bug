# Open threads and deferred decisions

This document captures design questions that emerged during planning but were not resolved in the ADRs, decisions that were made in conversation but not yet reflected in the design docs, and things we discussed but explicitly deferred.

---

## Phase 1 decisions that diverge from design-auth.md

These were decided during implementation planning and override the text in `design-auth.md`:

**Sliding window session refresh is in Phase 1.**
`design-auth.md` says "Refresh is not implemented in Phase 1". We decided against the simpler fixed-expiry approach: the sliding window is cheap (re-issue the JWT on every valid request) and avoids asking users to re-authenticate mid-session. goth does not manage git-bug session JWTs — the sliding window is purely in git-bug's `JWTMiddleware`.

**OIDC auto-discovery is in Phase 1.**
`design-auth.md` only mentions GitHub and Google. We added the `openid-connect` goth provider (which reads `/.well-known/openid-configuration`) so operators can plug in any OIDC-compliant IdP via `--oidc-discovery-url` without a code change.

**No `/auth/user` HTTP route.**
`design-auth.md` lists `GET /auth/user` as a route for "return current user info (for webui header display)". This is unnecessary: the `repository { userIdentity { ... } }` GraphQL field already returns the authenticated user once `JWTMiddleware` injects the identity into context. The webui's `useAuth()` hook already queries that field.

---

## Unresolved design questions

### Q1: Bridge External Account Reference conflict

The GitHub bridge currently stores the GitHub *login* (mutable username) as immutable metadata under a different key than the OAuth format. If a user was imported by the bridge and then logs in via OAuth for the first time, `ResolveIdentityImmutableMetadata("github:user-id", numericID)` will not find the bridge-imported identity — they get a duplicate.

Options: dual-lookup during OAuth callback (check `"github:user-id"` first, then fall back to bridge's key), a one-time migration command, or fix the bridge to write `"github:user-id"` alongside its existing key going forward. No decision made.

This must be resolved before a deployment that has both bridge-imported identities and OAuth login — which is the common case.

### Q3: Multi-provider identity linking

If the same person authenticates via GitHub, then via Google, they get two separate git-bug identities. There is no "link accounts" flow. Which identity would be primary? Can they be merged? No design exists for this. Deferred — not blocking Phase 1 but worth deciding before Phase 1 is widely deployed.

### Q4: ProjectConfig bootstrap authority

The first `DeclareAdminOp` in ProjectConfig is unsigned — no admin exists yet to sign it. Any client can verify the chain forward from that point, but cannot cryptographically verify the bootstrap admin is legitimate. Is trusting whoever seeded the repo acceptable, or is there a stronger mechanism (e.g. the ServerIdentity signs its own bootstrap, provable via the ServerIdentity's key chain)?

### Q5: ServerIdentity key model — shared vs. per-user

Phase 3 describes the ServerIdentity signing operations attributed to web users with unprotected identities. For protected-identity users who add the server's public key to their identity, the current design implies one ServerIdentity key signing for all such users. This means any compromise of the server key gives impersonation power over all consenting protected users simultaneously. A per-user server keypair avoids this but adds operational complexity. No decision.

### Q7: Key revocation propagation — caching strategy

If a user revokes the server key (removes it from their identity by pushing a new version), the server must notice before the next signed operation. If the server caches identity key sets indefinitely in `RepoCacheIdentity`, it will keep signing with a key the user has already revoked. The caching and invalidation strategy for identity key sets is not specified. This needs a decision before Phase 3 ships.

---

## ADR-0002: signer interface shape — RESOLVED

Implemented in Phase 2. The interface is:

```go
// repository/signer.go
type Signer interface {
    Sign(payload []byte) (signature []byte, err error)
}
```

`SignatureFormat` is NOT a return value — the format is implicit in the returned bytes (PGP armored vs SSHSIG armored header), and the verification path detects it via `repository.IsSSHSignature(sig)`. Concrete implementations: `SSHAgentSigner` (dials `$SSH_AUTH_SOCK`) and `GPGSigner` (shells out to `gpg`). Tests inject `NewSSHAgentSignerWithAgent(pub, inMemoryAgent)` to avoid a real agent.

---

## atproto / goth feasibility spike (deferred, needed before Phase 5)

We noted during planning that atproto OAuth is materially different from what goth's `Provider` interface assumes:

- No pre-registered client secret — client ID is a URL, not an opaque string.
- DPoP: each token exchange and API call requires an ephemeral keypair and a signed proof-of-possession header. `goth` calls `oauth2.Exchange` internally and does not expose a hook to inject DPoP headers.
- Authorization server discovery is per-user (resolve handle → DID → PDS endpoint), not a single static `.well-known` URL.

A custom `goth.Provider` can override the flow, but it would need to bypass `oauth2.Exchange` entirely and drive the HTTP exchange manually. This is doable but non-trivial.

**Before committing to Phase 5**, a focused spike is needed to answer: can a custom `goth.Provider` accommodate atproto's DPoP requirements without forking goth, or does atproto auth need to live outside the goth abstraction entirely?

---

## Bridge numeric user ID gap (independent fix, blocks identity unification)

The GitHub bridge imports identities with the login (username) as the lookup key, not the numeric user ID. This has two problems:

1. Username changes break the lookup — the bridge will create a duplicate identity for the same person after a rename.
2. OAuth and bridge imports use different keys, preventing identity unification (see Q1 above).

Fix: in the bridge's `ensurePerson`, also write `"github:user-id": <numericID>` as immutable metadata. This is independent of all auth phases and can be done any time. It is a prerequisite for Q1 having a clean answer.

---

## Implementation state at time of writing

- `docs/` (all ADRs, `design-auth.md`, `prd-external-auth.md`): on disk, untracked.
- `api/auth/jwt.go`, `api/auth/cookie.go`: on disk, untracked.
- `api/auth/oauth/` (handler, import, tests): on disk, untracked.
- `api/auth/middleware_test.go`: on disk, untracked.
- Phase 1 wiring (`JWTMiddleware` in `middleware.go`, OAuth routes in `commands/webui.go`, `goth` and `golang-jwt/jwt/v5` dependencies): in a git stash. `go.mod`/`go.sum` not yet updated.

**Phase 2 key format change is complete (trunk branch):**

- `repository/signer.go`: `Signer` interface, `SSHAgentSigner`, `GPGSigner`, SSHSIG encode/decode/verify
- `entities/identity/key.go`: rewritten — `Key{publicKeyMultibase, pgpEntity, origin}` replacing PGP struct; `NewSSHKey`, `NewGPGKey`, `NewKeyWithSigner` constructors; `Key.Signer()` method
- `entities/identity/version.go`: `formatVersion` bumped 2 → 3
- `entities/identity/interface.go`: `Signer() (repository.Signer, error)` replaces `SigningKey(repo RepoKeyring) (*Key, error)`
- `repository/repo.go`: `Commit.Signature`/`SignedData` are now `[]byte`; `StoreSignedCommit` takes `Signer`
- `repository/gogit.go`, `repository/mock_repo.go`: updated accordingly
- `entity/dag/operation_pack.go`: signing path uses `identity.Signer()`; verification dispatches on SSHSIG vs PGP header

Phase 1 stash can be applied on top of this.
