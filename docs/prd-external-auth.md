# PRD: External Authentication for Public Webui

## Problem Statement

git-bug's webui currently only supports a single hard-coded user: the local git config identity of whoever is running the server. This makes it impossible to host the webui publicly and allow external collaborators to interact with bugs under their own identity. Anyone who opens the webui either gets read-only access or acts as the server operator — there is no way for a third party to log in, be identified, and have their actions attributed to them.

## Solution

Add OAuth/OIDC external authentication to the webui so that any user with a supported provider account (initially GitHub and Google) can log in, be mapped to a git-bug Identity, and create operations attributed to that Identity. The webui becomes hostable as a public service where collaborators interact under their real identities without sharing server credentials.

This PRD covers Phase 1 of a five-phase roadmap (see ADR-0004). Phase 1 ships a working, deployable external auth flow. Later phases add cryptographic signing of web operations, key import, and atproto/Bluesky support.

## User Stories

### Operator stories

1. As a repository owner, I want to run `git bug webui` with provider credentials as flags or environment variables, so that I can host the webui publicly without any interactive setup ceremony.
2. As an operator, I want GitHub and Google OAuth to work out of the box, so that my collaborators can log in immediately without registering custom providers.
3. As an operator, I want the server to auto-generate a JWT signing secret at startup, so that I get a working session system with zero configuration.
4. As an operator running multiple instances behind a load balancer, I want to supply a stable `--auth-secret` flag or `GIT_BUG_AUTH_SECRET` environment variable, so that sessions survive across instances and restarts.
5. As an operator, I want read-only mode to remain accessible without authentication, so that public visitors can browse bugs without logging in.
6. As an operator, I want authenticated write operations to fail clearly if the session is invalid or expired, so that users get a usable error rather than silent data corruption.
7. As an operator, I want the webui to remain fully functional for a single local user (the existing mode) when no OAuth providers are configured, so that I do not break existing deployments.

### Collaborator (web user) stories

8. As a collaborator, I want to click "Sign in with GitHub" or "Sign in with Google" and be redirected through the provider's OAuth flow, so that I never have to create a git-bug-specific account.
9. As a collaborator, I want my first login to automatically create a git-bug Identity linked to my provider account, so that my operations are attributed to me without manual setup.
10. As a collaborator, I want subsequent logins to recover my existing Identity, so that all my past and future operations are attributed to the same person.
11. As a collaborator, I want my display name and avatar from the OAuth provider to appear in the webui, so that other participants can recognise me.
12. As a collaborator, I want to be able to log out and have my session invalidated, so that I can safely use the webui on shared machines.
13. As a collaborator, I want the webui to show me who I am logged in as, so that I can confirm my Identity before making changes.
14. As a collaborator, I want my session to expire after a reasonable time, so that I am not permanently authenticated on devices I no longer control.

### Local user (existing workflow) stories

15. As a local git-bug user, I want to run `git bug webui` without any OAuth configuration and have it work exactly as before, so that external auth is strictly additive.
16. As a local user who later hosts the webui publicly, I want external collaborators' OAuth-created Identities to coexist with my local Identity in the same repo, so that all participants are represented correctly.

## Implementation Decisions

### OAuth/OIDC library
Use `markbates/goth` for the OAuth/OIDC integration. It provides a unified `Provider` and `Session` interface across GitHub (OAuth 2.0, non-OIDC) and Google (OIDC), and its interface is extensible for future atproto support (Phase 5).

### Provider configuration
Provider credentials (client ID and client secret) are supplied via `git bug webui` flags or environment variables. No subcommand or interactive configuration. Example:

```
--github-client-id / GIT_BUG_GITHUB_CLIENT_ID
--github-client-secret / GIT_BUG_GITHUB_CLIENT_SECRET
--google-client-id / GIT_BUG_GOOGLE_CLIENT_ID
--google-client-secret / GIT_BUG_GOOGLE_CLIENT_SECRET
```

If no provider is configured, external auth routes are not registered and the server runs in single-user mode as today.

### Session management
Sessions use short-lived server-issued JWTs. The JWT payload contains the git-bug `entity.Id` of the authenticated Identity and an expiry. The JWT is delivered as an HTTP-only cookie.

The signing secret is auto-generated in memory at startup (restart invalidates all sessions). Operators supply `--auth-secret` / `GIT_BUG_AUTH_SECRET` for persistence or load-balanced deployments.

The existing `api/auth` package is extended: `Middleware` gains a new variant that resolves the session cookie to an `entity.Id` rather than using the fixed single-user ID.

### Identity import and lookup
On every OAuth callback:

1. Extract the provider's stable numeric user ID from the `goth.User` struct.
2. Call `ResolveIdentityImmutableMetadata("<provider>:user-id", "<id>")` on the cache.
3. If found: use the existing Identity.
4. If not found: create a new Identity via `repo.Identities().NewRaw(name, email, login, avatarURL, nil, map[string]string{"<provider>:user-id": "<id>"})` and commit it.

The External Account Reference key format is `"<provider>:user-id"` (e.g. `"github:user-id"`). The value is always the provider's opaque numeric ID, never the mutable username.

In Phase 1 the Identity's `keys` field is left nil. Identities created by external auth are unprotected. Operations they author are unsigned. This is intentional — see ADR-0004.

### Adoption of existing Identity (FinishConfig pattern)
On first login via a provider, if no Identity matches the External Account Reference but a default user Identity is set (single-user mode, local git config user), offer to link the external account to that Identity by adding the External Account Reference as metadata. This follows the existing `bridge/core.FinishConfig` pattern.

### GraphQL API
The existing `api/auth` context helpers (`CtxWithUser`, `UserFromCtx`) are unchanged. The middleware layer sets the Identity in context; the GraphQL resolvers are unaffected.

A small set of new GraphQL fields or a REST endpoint is needed to support the webui login/logout flow and to expose the current user to the frontend. The exact shape is deferred to implementation.

### Routing
New HTTP routes registered on the gorilla/mux router when at least one provider is configured:

- `GET /auth/{provider}` — initiate OAuth redirect
- `GET /auth/{provider}/callback` — OAuth callback, sets session cookie, redirects to webui
- `POST /auth/logout` — clears session cookie

### No key management in Phase 1
The `Key` type, `publicKeyMultibase`, `pgpEntity`, and operation commit signing changes described in ADR-0001, ADR-0002, and ADR-0003 are explicitly out of scope for this phase. The `keys` field on imported Identities is nil. Unsigned operations from web users are accepted — there is no verification failure because the Identity has no keys.

## Testing Decisions

Good tests for this feature verify externally observable behavior: does the correct Identity appear in the operation context after login, does an invalid session correctly fail, does a first-time login create an Identity with the right metadata? Tests should not assert on JWT internals, session cookie format, or goth's internal state.

### What to test

**Session middleware**
Given a valid JWT cookie containing a known `entity.Id`, the middleware must place that Identity in the request context. Given an expired, malformed, or absent cookie, `UserFromCtx` must return `ErrNotAuthenticated`. Prior art: `api/auth/middleware.go` and its existing test if present.

**Identity import logic**
Given a `goth.User` from a provider, the import function must:
- Return an existing Identity when `ResolveIdentityImmutableMetadata` matches.
- Create a new Identity with correct name, email, login, avatarURL, and immutable metadata when no match exists.
- Set the External Account Reference to the provider's numeric user ID, not the username.
Prior art: `cache/identity_subcache.go` — `ResolveIdentityImmutableMetadata` is already tested indirectly by bridge import tests.

**OAuth routes (integration)**
The callback handler must set a valid session cookie and redirect on success, and return an appropriate error on provider failure. These tests mock the `goth.Provider` interface.

**Single-user mode non-regression**
When no provider is configured, the existing fixed-identity middleware path must behave identically to today. Prior art: existing webui handler tests in `webui/handler_test.go`.

### Modules with new tests
- `api/auth` — session JWT creation, validation, expiry, middleware resolution
- A new `api/auth/oauth` (or similar) package — identity import/lookup logic, callback handler

### What not to test
- goth internals or provider-specific OAuth wire formats
- JWT encoding details (test the behavior the token enables, not the token itself)
- The exact shape of the session cookie (test that it is set, not how it is encoded)

## Out of Scope

The following are explicitly deferred to later phases per ADR-0004:

- **Phase 2:** `publicKeyMultibase` key format change, importing provider public keys (GitHub GPG keys API), `git bug user key generate`, `git verify-commit` interoperability.
- **Phase 3:** ServerIdentity, AdminRole, cryptographically signed web operations, `git bug webui setup`.
- **Phase 4:** ProjectConfig entity, distributed role management, AdminIdentity declarations in the git object chain.
- **Phase 5:** atproto/Bluesky OAuth, DPoP, PLC directory key tracking, go-did-it integration.
- **All phases:** X.509 signing, SAML, LDAP, or any provider not expressly listed above.
- Key-based Identity adoption (challenge-response with an existing private key) — this requires Phase 2 key infrastructure.

## Further Notes

The five-phase roadmap and the rationale for each architectural decision are documented in:
- `docs/adr/0001` — publicKeyMultibase key format
- `docs/adr/0002` — operation commit signing format
- `docs/adr/0003` — ServerIdentity and AdminRole impersonation
- `docs/adr/0004` — external auth phasing

The domain vocabulary used throughout this PRD is defined in `CONTEXT.md` at the repository root.

PR #1527 (external, not merged) addresses a related bug where git-bug's OpenPGP public key serialization is broken. That fix is superseded by the Phase 2 key format change (ADR-0001) and should not be merged independently.
