# ADR 0001: Public key storage format — publicKeyMultibase over OpenPGP armored

## Status
Accepted

## Context

git-bug Identities carry public keys used to verify that operation commits were authored by the claimed Identity. The current implementation stores only `*packet.PublicKey` from the OpenPGP library — a partial OpenPGP structure that cannot be imported into GPG (`gpg --import` rejects it because it lacks a User ID and self-signed certification). This means `git verify-commit` does not work on git-bug operation commits today (see PR #1527).

Three forces push toward changing the format:

1. **Algorithm lock-in.** OpenPGP armored format only represents OpenPGP keys (RSA by default). Modern signing uses Ed25519, and DID-based identities (atproto/Bluesky) use Ed25519 or P-256. There is no OpenPGP path to store or verify these.

2. **DID incompatibility.** W3C DID documents express public keys as `publicKeyMultibase` — a self-describing, algorithm-agnostic encoding. Importing a user's DID keys into a git-bug Identity requires a format that can represent any algorithm.

3. **The current format is unused.** No UX exists to generate keys and attach them to an Identity. The format has zero deployed data, making this the lowest-cost moment to change it.

The alternative considered was staying with OpenPGP and fixing the serialization (storing the full GPG entity as PR #1527 proposes). This resolves the `git verify-commit` problem for GPG users but does not address algorithm agnosticism or DID compatibility, and produces much larger stored keys (~3× the size of the minimal multibase encoding).

## Decision

Public keys in Identity versions are stored as `publicKeyMultibase`: a base58btc-encoded string with a multicodec varint prefix identifying the algorithm, followed by the raw key bytes. This is the W3C DID `publicKeyMultibase` format, implemented by `go-did-it`.

For GPG-origin keys (imported from a user's GPG keyring or from a provider's GPG keys API), the full armored GPG entity is also stored in a `pgpEntity` field alongside `publicKeyMultibase`. This preserves the User ID and self-signed certification needed for `gpg --import`. `publicKeyMultibase` is derived from the entity — the entity is authoritative for the key bytes, and the two fields are always written together atomically.

For SSH-origin keys, `publicKeyMultibase` is sufficient. The SSH public key wire format (algorithm label + length-prefixed key bytes + optional comment) is fully reconstructable from the multibase representation and algorithm. No extra field is needed.

The default algorithm for newly generated keys is Ed25519. The `formatVersion` in identity version data is bumped from 2 to 3.

Private keys continue to be held in the OS keyring (via `99designs/keyring`) or delegated to an SSH or GPG agent. They are never stored in git objects.

## Consequences

- Any client on format version 3 can represent Ed25519, P-256, P-384, RSA, secp256k1, and other algorithm keys uniformly.
- Keys imported from DID documents (atproto, `did:plc`) can be stored directly without conversion.
- SSH Ed25519 keys (from `user.signingkey` git config or provider APIs) can be stored as Ed25519 multibase — the same bytes, different encoding.
- GPG users retain full compatibility: the `pgpEntity` field enables round-trip through the GPG keyring.
- The `PGPEntity()` method on `Key` is removed. Code that calls it (operation signing and verification in `operation_pack.go`, `StoreSignedCommit` in `repository/gogit.go`) must be updated to use algorithm-appropriate signing. See ADR-0002.
- No migration is needed for existing data because no deployed identity currently has keys attached (the key generation UX does not exist yet).
