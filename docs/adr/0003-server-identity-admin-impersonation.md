# ADR 0003: Server Identity and AdminRole impersonation

## Status
Accepted

## Context

When a user interacts with a public git-bug webui, operations (bug comments, status changes, etc.) must be attributed to the user's git-bug Identity and committed to git. For operations to be cryptographically signed, the signing key must be available at the point of commit — on the server.

Three approaches were considered:

**A. Per-user server keypair embedded in Identity.** The server generates a keypair per user, adds the public key to the user's Identity (signed by the user's existing key or as the first key if unprotected), and signs operations with the private key. Benefit: explicit per-user authorization. Drawback: requires touching every user Identity; server key rotation requires updating every user Identity that embeds it.

**B. Per-user server keypair without Identity embedding.** Server holds a keypair per user but the public key is not in the Identity. Drawback: no verifiable proof — any client pulling the repo cannot verify the operations without trusting the server's word. Rejected for a distributed system.

**C. Server as a real git-bug Identity with AdminRole.** The server has its own Identity stored in the repo. That Identity is declared as AdminIdentity in ProjectConfig. An AdminIdentity may sign operations attributed to any *unprotected* Identity. Key management (rotation, multiple keys for multi-instance deployments) is handled by the existing Identity versioning and multi-key infrastructure — no new code.

The key distinction in approach C: AdminRole applies **only to unprotected Identities** (Identities with no keys in their `keys` field). A user who has set up their own keys retains exclusive signing authority over their operations. Admin impersonation cannot bypass a protected Identity's key chain.

This boundary is the critical security property: opting into cryptographic key protection means the admin cannot act on your behalf.

## Decision

The server is represented as a git-bug Identity (ServerIdentity), created during `git bug webui setup` and stored in the repo alongside user Identities. The ServerIdentity is declared as an AdminIdentity in ProjectConfig.

Verification in `operation_pack.go` gains one additional rule after the existing key check:

```
if author has keys → commit must be signed by one of author's valid keys (unchanged)
if author has no keys AND commit is signed by a declared AdminIdentity → accepted
if author has no keys AND commit is unsigned → accepted (no cryptographic guarantee)
```

The third rule preserves backward compatibility: unsigned commits from unprotected identities remain valid (as they are today and in Phase 1 of external auth). The second rule adds a new positive-verification path for server-signed operations.

The ServerIdentity uses the standard Identity multi-key model. Each server instance can hold a separate key in the ServerIdentity (for load-balanced deployments). Key rotation is a new Identity version signed by the previous key — identical to any other Identity rotation.

AdminRole is also independently useful beyond the webui: bridge import tools and CI/CD bots can use it to attribute operations to the correct imported user Identities without those users needing to have run git-bug locally.

ProjectConfig (a future entity, see CONTEXT.md) is the authoritative source for AdminIdentity declarations. Until ProjectConfig is implemented, the ServerIdentity declaration lives in the repo's local git config as a transitional measure.

## Consequences

- Web users with unprotected Identities get cryptographically signed, verifiable operations with zero extra setup.
- Users with protected Identities (their own keys) are unaffected — their operations must be signed with their own key; the server cannot impersonate them.
- Server key rotation does not require touching any user Identity.
- Multi-instance deployments add keys to the ServerIdentity rather than maintaining per-user key sets.
- The AdminRole concept is available as a general feature: any Identity can be granted admin rights for automated tooling.
- This design requires ProjectConfig to be implemented before AdminRole is fully distributed and verifiable by all clients. Until then, admin declarations are local-config only (Phase 3 of the external auth roadmap).
