# Design: Decoupled Identity Key Registry

## Status

Proposed

## Related artefacts

- `CONTEXT.md` — domain glossary; all terms used here are defined there
- `docs/design-auth.md` — master auth plan; this document extends Phase 2 (key management)
- `docs/adr/0001` — publicKeyMultibase key storage format
- `docs/adr/0002` — operation commit signing (SSH / PGP)
- `docs/adr/0004` — phasing rationale

---

## Background: why key validity at logical time matters

git-bug stores entity operations (bugs, PRs, etc.) as git commits in a DAG. Each operation pack is signed by the author's Identity key so that, when a collaborator receives the pack, they can verify it was genuinely authored by the claimed Identity and not tampered with.

Because Identities are mutable — users add devices, rotate keys, revoke compromised keys — the key set valid for a given operation is not necessarily the *current* key set. It is the key set that was valid *when the operation was created*. Verifying a three-year-old comment requires knowing which keys that Identity held three years ago.

This is the job of `ValidKeysAtTime(clockName string, time lamport.Time) []*Key`. The "time" here is a **Lamport clock value**, not wall-clock time, because the CRDT is a distributed system with no trusted global clock. Lamport clocks provide causal ordering across concurrent writes without any clock synchronisation.

The current implementation satisfies this by embedding Lamport clock snapshots directly in identity versions: when a user commits a new identity version (e.g. adding a key), that version records the current Lamport time of every known entity namespace. `ValidKeysAtTime` walks the version chain and returns the key set from the most recent version whose recorded clock is ≤ the query time.

---

## Problem statement

Three structural weaknesses in the current model compound as key management becomes a real feature.

**1. Keys are per-repo.**
Identity versions, including their key sets, live in git refs inside a single repository. When a user rotates a key, that change must be manually pushed to every repo they participate in. This is tolerable while the key-management UX does not exist, but becomes unworkable the moment key rotation is a routine operation: a user with five repos must remember to propagate every key change to all five.

**2. Identity is a linear chain.**
Identity versions are enforced as a strictly linear commit chain. This is required today because the key set is security-critical state that must have a canonical ordering — two concurrent edits cannot be merged without knowing which wins. The consequence is that any two devices editing the same identity simultaneously (changing a display name, adding a key from a second device) produce a fork that cannot be automatically resolved. The network splits until someone manually reconciles the chain.

**3. Key validity is bounded by self-reported Lamport time.**
The Lamport clock value recorded in an operation pack is written by the operation *author*. An attacker who holds a rotated-out key K1 can create a new DAG branch forking from an old commit, write an operation pack claiming a Lamport time within K1's former validity window, and have that pack pass verification. The Lamport monotonicity property (you cannot claim a time lower than your causal ancestors) partially limits this: forking from a recent branch point is hard because the clock has advanced. But forking from an old branch point is always possible, and on a quiet entity the constraint is weak.

---

## Goals

### Functional goals

1. **Cross-repo identity.** Key changes propagate automatically to all repositories that share a contributor — no manual push required per repo.

2. **Mergeable identity.** Two devices can edit identity concurrently and the results merge cleanly, with no manual reconciliation and no risk of a permanent network split.

3. **No new keys for users.** Users manage identity through their existing git signing infrastructure (SSH agent, GPG keyring). git-bug bridges those keys into the identity system; no separate git-bug key type is introduced.

4. **Backward compatibility.** All existing identities continue to work with no migration required. The new model is an opt-in extension, not a replacement.

5. **Cross-repo identity organisation.** A shared git-bug instance (a community server, a company server) can act as a natural identity registry for its users, eliminating per-repo identity propagation for that community entirely.

6. **Pluggable identity backends.** The system is not coupled to any specific external protocol. DID-PLC, a plain git repository, a community git-bug instance — all are valid backends implementing the same interface.

### Security goals

7. **Preserve `ValidKeysAtTime` semantics.** The Lamport-time-based key validity check must continue to work correctly across all backends and for all existing identities.

8. **Improve fork-attack detectability.** While the fork attack cannot be fully closed without a trusted external clock, the design should make forged operations more detectable and raise the cost of the attack.

9. **Key revocation is visible across repos.** After a key is revoked, all repos that resolve that identity pick up the revocation automatically — not just the repo where the revocation was committed.

---

## Security model

### Active key compromise

If the key currently in use is compromised, the attacker can sign arbitrary operations that pass verification. This is true in the current model and in every model that uses static signing keys. It is the baseline failure mode for all public-key authentication systems.

The only cryptographic remedy is **forward-secure signatures** (e.g. XMSS): a key-evolving scheme where the signing key advances monotonically and past key material is irrecoverably deleted after each use. An attacker who compromises the current key cannot sign for any past time slot.

Forward-secure signatures are **not compatible with existing git signing keys** (SSH, GPG). Those keys are stateless — the same private key signs any message at any time. Forward security requires the signer to maintain and advance state (a leaf index in a Merkle tree), which the SSH agent and GPG have no concept of. Multi-device use compounds the problem: two devices sharing an identity cannot advance a single key schedule concurrently without synchronisation.

Forward security is therefore not a goal of this design. The accepted mitigations are short key lifetimes and prompt revocation.

### Post-rotation compromise

If a key K1 is rotated out (replaced by K2) and then K1 is compromised, the attacker cannot forge new operations that appear to have been created after the rotation — those would need to pass `ValidKeysAtTime` with a Lamport time in K2's validity range, which requires K2.

The attacker can forge operations that appear to have been created *before* the rotation, by creating a new DAG branch forking from a point where the clock was in K1's validity range. This is the **Lamport fork attack**.

**The Lamport monotonicity partial mitigation.** A new branch must have Lamport time ≥ the max of its causal ancestors. If the entity's clock has advanced significantly past the rotation point, forking from before the rotation requires branching off a much older commit, making the forged branch visibly anachronistic relative to the entity's real history. This is not a hard constraint — you can always fork from an arbitrarily old commit — but it raises the cost and visibility of the attack. It is a mitigation, not a solution.

**The equivalence of both models.** This mitigation exists equally in the current model and in the proposed design. Neither model closes the fork attack. The design goal is to improve detectability, not to eliminate the attack.

### Key validity is controlled by the identity owner, not the operation author

This is the critical invariant that must be preserved across all design choices.

In the current model, the key-validity window is defined by the identity's version chain — content-addressed in git, immutable once committed. The operation author records a Lamport time; the identity chain determines which key set maps to that time. The operation author cannot choose the mapping.

A naive alternative — embedding a key-log anchor (e.g. a DID-PLC CID) directly in each operation pack — would violate this invariant: the operation author would be choosing which key-set snapshot to present for their own verification. An attacker with a compromised K1 would simply embed the CID from before the rotation, and the operation would verify successfully regardless of the actual rotation time.

The design must keep key-validity mapping under identity-owner control. Operation authors record logical time (Lamport clocks); the identity chain maps logical time to key sets; the operation author cannot influence that mapping.

---

## Design alternatives considered

### A: Embed a key-log anchor (CID) in each operation pack

**Rejected.** Inverts the control relationship — the operation author chooses the key snapshot used to verify their own operations. An attacker with a compromised key selects an anchor predating the rotation. See the security model section above.

### B: Use DID-PLC as a drop-in replacement for the local identity

**Rejected.** DID-PLC's audit log is indexed by wall-clock time (server-stamped). `ValidKeysAtTime` requires logical time (Lamport clocks). There is no reliable, trustworthy bridge between the two unless the identity owner explicitly records the mapping — which is exactly what the native version chain already does. Using DID-PLC as a pure replacement eliminates the Lamport anchoring mechanism and leaves no way to answer "which keys were valid at Lamport time T?"

### C: DAG-ancestry-based key validity

**Interesting but rejected for this design.** Instead of Lamport clocks, key validity could be defined by git DAG ancestry: K1 is valid for commits that are not causally downstream of the K1→K2 rotation commit. This is content-addressed and unforgeable — an operation either has the rotation commit as an ancestor or it doesn't.

The obstacle is the partial-copy property: git-bug entities are designed to be independently clonable (a partial clone of bugs without PRs, or without identities, must work). DAG-ancestry-based key validity would require cross-entity DAG references that break partial copy. It also requires a "rotation seal" operation per entity to definitively close the old key's window, which at scale (thousands of bugs) is impractical.

The Lamport clock approach, despite its fork-attack weakness, fits the partial-copy model naturally: clock values are just integers recorded in each pack, not cross-entity DAG links.

### D: Per-entity revocation seals

**Rejected.** Sealing the old key's validity window definitively requires posting a signed operation on each entity (each bug, each PR) asserting "K1 is invalid from this point forward." For a user with thousands of entities this is completely impractical.

The cross-namespace vector clock (see below) provides the same detection benefit as a seal without requiring any per-entity action.

### E: RFC 3161 signed timestamps

**Partially adopted.** RFC 3161 timestamps — cryptographic proofs from a trusted timestamp authority that a document existed at wall-clock time T — would allow comparing operation creation time against key rotation time reliably, bypassing the Lamport clock entirely.

The full form (embedding an RFC 3161 token in every operation pack) is not adopted: it requires online access at operation-creation time and adds a dependency on a timestamp authority.

The partial form is captured in the PLCKeyLog backend: DID-PLC provides server-stamped `createdAt` timestamps on each key operation, which git-bug uses as a secondary consistency check when the backend is PLC (see below).

### F: Any external append-only log (the adopted approach)

The requirements on an external key system, derived from first principles, are:

1. Unique identity naming
2. Append-only history (past changes are not retroactively modifiable)
3. Linear ordering (entries are causally ordered; each references the previous)
4. Authorized writes (only the identity owner can append)
5. Public read access (verifiers can fetch without authentication)

DID-PLC satisfies all five. But so does a plain public git repository with a protected branch. So does another git-bug instance acting as a community identity server. So does any HTTP append-only log.

The design therefore defines a **KeyLog interface** that any of these implements, and makes the backend pluggable. DID-PLC is one implementation. The system is not coupled to it.

---

## Design

### The KeyLog interface

A **KeyLog** is an append-only, publicly-readable, authorized log of key-set changes for one identity. It is the single source of truth for *what keys exist*. The local identity CRDT remains the single source of truth for *when those keys became effective* (Lamport time).

```go
type KeyLog interface {
    // KeysAt returns the accumulated key set at the given anchor —
    // the result of replaying all Add/Remove operations from genesis
    // through and including the entry at anchor. Implementations cache results.
    KeysAt(anchor string) ([]*Key, error)

    // Append submits a key-update operation. Each backend is initialized
    // with its own signing mechanism (see backends); the interface does not
    // carry a signer. Returns the new anchor and its sequence number.
    Append(op KeyOp) (anchor string, seq uint64, err error)

    // Parent returns the anchor and sequence number of the entry preceding
    // the given anchor, or ("", 0, nil) at genesis.
    Parent(anchor string) (string, uint64, error)
}

// KeyOp is an atomic change to the key set.
type KeyOp struct {
    Add    []*Key
    Remove []*Key
}
```

**Signing is backend-local.** The same underlying key (SSH Ed25519, GPG, etc.) signs everything: git commits, operation packs, and KeyLog updates. Sharing key infrastructure across all three layers is an explicit goal — users manage one key, not one per system.

What differs between backends is **signature format**, not key material. `repository.Signer` produces SSHSIG or PGP armored output suited for git's `gpgsig` commit header. PLC operations require a compact signature over the operation JSON in PLC's native format — same Ed25519 bytes underneath, different wire encoding. Each backend is therefore initialized with the user's current key (from the SSH agent or GPG keyring) and is responsible for producing its own format. The `KeyLog` interface carries no signer; signing is an initialization-time concern of each backend.

**Sequence numbers** allow O(1) merge tie-breaking without network access during merge. Each identity CRDT version stores `keylog_seq` alongside `keylog_anchor`. When merging two concurrent versions that reference different anchors on the same log, the version with the higher `keylog_seq` wins. For `GitKeyLog` the sequence number is the commit depth from genesis; for `PLCKeyLog` it is the PLC `seq` field present in every operation.

**The identity ID** (derived from the genesis content hash, as today) is stable across KeyLog backend changes. The KeyLog URL is a resolution hint stored in the identity CRDT's first version — not part of the canonical identity identifier. An identity can migrate between backends without changing its ID.

### KeyLog backends

**Inline (existing identities).**
Keys are stored directly in identity CRDT versions, as in the current v3 format. No `keylog_url` or `keylog_anchor` field in the version JSON; absence of `keylog_url` is the discriminator. `ValidKeysAtTime` uses the existing inline path unchanged. The `KeyLog` interface is not involved. This is the backward-compatibility path: all existing identities work without any change.

This backend also serves as the path for **cross-repo identity adoption**: a user who already has an inline identity in repo A and wants to participate in repo B simply pushes their identity ref to repo B, exactly as today. The KeyLog design does not break or replace this workflow — it extends it for users who want cross-repo key management.

**`GitKeyLog` — public git repo with a protected branch.**
Each key-set change is a commit on the log branch, signed with the current key. The commit hash is the anchor; commit depth from genesis is the sequence number. Content-addressed and self-hostable; works with any git hosting service.

Branch protection (`--no-force-push`, no deletion) provides the append-only guarantee. This is a **policy guarantee**, not a cryptographic one: the hosting provider or a compromised admin can rewrite the log silently. Individual entry commit hashes are content-addressed (a known anchor can be checked for presence), but the branch tip is mutable by the provider. Operators who need a stronger guarantee should use PLCKeyLog. See the security properties table.

`GitKeyLog` supports an optional **recovery key** (an Ed25519 key kept offline, whose public key is recorded in the genesis entry with `role: recovery`). The recovery key can authorize a key-set update when all signing keys are lost. Without a recovery key, losing all signing keys permanently freezes the log — an accepted limitation that must be planned for at setup time.

A `GitKeyLog` backend can itself be a **git-bug repository** (a community instance or company server). This makes a shared git-bug deployment a natural identity server for its community: users on that instance get cross-repo key management with no dedicated infrastructure. Their inline identities on the shared instance effectively serve as that community's identity registry.

**`PLCKeyLog` — DID-PLC server.**
The anchor is the IPLD CID of the PLC operation; the sequence number is the PLC `seq` field. Provides interoperability with the atproto/Bluesky ecosystem. The PLC server's `createdAt` timestamp is an authoritative wall-clock anchor usable as a secondary validity check (see below). PLC's native recovery key mechanism applies.

**Other implementations.** Any service that satisfies the five structural requirements (unique naming, append-only, linear ordering, authorized writes, public read) is a valid backend. The interface is the contract; the choice of backend is a deployment decision.

### Local identity CRDT becomes a mergeable DAG

With keys living in the KeyLog, the local identity CRDT no longer holds security-critical state that requires canonical ordering. Each version holds:

| Field | Notes |
|---|---|
| `keylog_url` | Resolution hint (immutable, set at genesis; absent for inline identities) |
| `keylog_anchor` | Opaque anchor string at commit time |
| `keylog_seq` | Sequence number at that anchor — enables O(1) merge tie-breaking |
| `times` | Lamport clocks of all namespaces at commit time (extended; see below) |
| `name`, `email`, `login`, `avatar_url` | Profile data (not security-critical) |

All these fields are safely mergeable:

- **Concurrent anchor commits** (two devices both recording after a key change): take the version with the higher `keylog_seq`. Both honestly reflect KeyLog state; higher sequence is more recent.
- **Concurrent profile edits**: last-write-wins on `unix_time`.
- **KeyLog linearity**: enforced by the backend. The local CRDT does not re-enforce it.

The identity DAG merges like any other entity. The network-split risk from concurrent edits disappears.

### Key propagation from existing git signing keys

Users manage zero new keys. git-bug bridges existing git signing keys into the KeyLog.

**Setup:**

```
git-bug identity create [--keylog <url>] [--recovery-key <pubkey>]
```

1. Reads the current signing key(s) from git config / ssh-agent / GPG keyring.
2. Constructs a genesis KeyLog entry with those public keys (plus recovery key if given).
3. Signs with the current key via the backend's own signer.
4. Submits to the KeyLog → receives `(anchor, seq)`.
5. Commits the local identity CRDT first version: `{ keylog_url, keylog_anchor, keylog_seq, times }`.
6. Advances the identity namespace Lamport clock.

**Adding a key** (`git-bug identity key add <pubkey>`):
Submits `KeyOp{ Add: [newKey] }` signed by any currently-valid key; records the new anchor, seq, and current Lamport clocks in a new CRDT version.

**Removing a key** (`git-bug identity key remove <pubkey>`):
Submits `KeyOp{ Remove: [revokedKey] }` signed by any *other* currently-valid key.

**Revocation is not queueable.** If the user is offline, `key remove` returns an error. A queued revocation is unsafe: operations created during the queue window would be signed by a key the user intends to invalidate. If the key is being revoked because it was compromised, offline queuing would leave a gap during which the attacker can forge operations that pass verification.

**Key addition while offline** can be queued safely (the new key is not usable until the KeyLog submission is confirmed). The queue is stored in `refs/git-bug/identity/pending` and flushed on next sync.

### Eager key loading — `ValidKeysAtTime` stays synchronous and error-free

`ValidKeysAtTime` is called synchronously during operation pack verification and has no error return on the `Interface`:

```go
ValidKeysAtTime(clockName string, time lamport.Time) []*Key  // unchanged
```

This interface must not change. The design resolves the apparent conflict with remote key fetches by **eager loading at identity read time**: when `identity.Read()` deserialises an identity from git, it immediately fetches and caches the key set for every distinct `keylog_anchor` found across all CRDT versions. Network access happens once at load time; `ValidKeysAtTime` hits the in-memory cache and remains synchronous and error-free.

`identity.Read()` already returns an error; the change is that the error may now include KeyLog fetch failures alongside git read failures. For inline identities the cache is populated from the inline keys in each version — no network call, behaviour unchanged.

### Cross-namespace vector clocks improve fork-attack detectability

Currently each operation pack records only its own entity namespace's Lamport clock. The proposal extends this to record the Lamport clocks of **all known namespaces** at write time, particularly the identity namespace.

The identity namespace Lamport clock advances whenever any user commits a new local identity CRDT version — i.e., every key change by any participant. This clock is shared across all users in the repository.

**Effect on the fork attack.** A forged operation using rotated-out key K1 must claim an identity Lamport time predating the rotation. But all legitimately-created operations after the rotation carry identity Lamport times ≥ the rotation point. An inconsistently-low identity clock in an incoming operation pack is a detectable anomaly at merge time.

**Important caveat.** This is a *detection mechanism*, not a preventive control. Its effectiveness depends on how frequently the identity clock advances. In a solo repository or a project where identity changes are rare, the identity clock may not advance for long periods, leaving the detection window wide. Cross-namespace clocks supplement the existing monotonicity constraint — they do not close the fork attack.

The benefit scales with community activity: in an active project with many contributors and regular key management operations, the identity clock advances frequently and makes the window very tight.

### Secondary wall-clock check (PLCKeyLog only)

When the backend is PLCKeyLog, the PLC entry at `keylog_anchor` carries a server-stamped `createdAt` timestamp. If an operation's `unix_time` is later than the PLC rotation timestamp but the operation uses a key the PLC had already rotated out, that is a detectable inconsistency — even though `unix_time` is self-reported. Combined with the cross-namespace identity clock check, an attacker must forge both a plausible `unix_time` and a consistent identity Lamport value to avoid detection.

### Backward compatibility and migration paths

- Identities without `keylog_url` use the existing inline key path. No field changes, no data migration, no breaking change for any existing v3 identity.
- `ValidKeysAtTime` callers are unaffected — the signature and semantics are identical.
- Cross-namespace Lamport clocks are additive. Older operation packs that record only their own namespace clock remain valid; cross-namespace detection is simply unavailable for those packs.
- A `git-bug identity migrate --keylog <url>` command can optionally register an existing inline identity with a KeyLog, creating log entries from the existing key history and updating the local CRDT to reference the log going forward.

### Cross-repo identity organisation

Because the KeyLog lives outside any single repository:

1. A contributor's identity CRDT first version (containing `keylog_url`) is pushed to each repo as today — a one-time step per repo.
2. Subsequent key changes update only the KeyLog. All repos that load this identity pick up the change on next sync automatically, with no per-repo push.
3. A GitKeyLog backend that is itself a git-bug repository turns that git-bug deployment into a community identity server. Users of that deployment get cross-repo key management with no dedicated infrastructure. Their inline identities on the shared instance serve as the identity registry for the community.

---

## Security properties

| Scenario | Inline (current) | GitKeyLog | PLCKeyLog |
|---|---|---|---|
| Active key compromised | breaks | breaks | breaks |
| Post-rotation compromise — fork attack | Lamport monotonicity partial mitigation | same + cross-namespace clock aids detection | same + PLC wall-clock timestamp aids detection |
| Concurrent edits from two devices | network split (linear chain) | auto-merge (DAG + seq) | auto-merge (DAG + seq) |
| History rewrite by attacker | impossible (content-addressed git) | possible by hosting provider (policy guarantee only, no cryptographic evidence) | impossible (content-addressed CIDs) |
| Key propagation across repos | manual per-repo push | automatic (KeyLog out-of-repo) | automatic |
| External dependency | none | git hosting provider | PLC server |
| Recovery when all signing keys lost | n/a | optional recovery key at genesis | PLC rotation key |
| Forward security | ✗ | ✗ | ✗ |
| DID ecosystem interoperability | ✗ | ✗ | ✓ |

**Forward security ceiling.** Static SSH and GPG keys cannot provide forward security — compromising the current key allows signing for any time. Forward-secure schemes (XMSS) require stateful signing incompatible with ssh-agent and multi-device use. Short key lifetimes, prompt revocation, and the detection mechanisms above are the practical mitigations within this constraint.

**GitKeyLog trust model.** The append-only guarantee rests on the git hosting provider's branch protection policy. A rogue admin can force-push and rewrite log history without leaving cryptographic evidence. Known commit hashes can be checked for presence, but the branch tip is mutable. Operators who need a cryptographic append-only guarantee should use PLCKeyLog.

**Key-validity control invariant.** In all backends, the mapping from Lamport time to key set is recorded by the identity owner in the local CRDT (the `keylog_anchor` per version). Operation authors record Lamport time but cannot choose the anchor — it is determined by the CRDT version the identity owner committed. This preserves the core security invariant: the operation author cannot choose the key set used to verify their own operations.

---

## Open questions

1. **KeyLog URL scheme.** Use the URL scheme to discriminate backend: `git+https://` / `git+ssh://` for GitKeyLog, `did:plc:` for PLCKeyLog. Alternatively, an explicit `keylog_backend` discriminator field alongside `keylog_url`. The URL-scheme approach avoids a new field but couples discriminator logic to URL parsing.

2. **Cross-namespace clock scope.** Recording all namespace clocks in every operation pack adds bytes proportional to the number of active entity types. A principled subset — always include the identity clock, include others opportunistically — may be sufficient. The identity clock is the only one that matters for key-validity detection.

3. **GitKeyLog log branch layout.** Dedicated repository (`github.com/alice/identity`) vs dedicated branch in an existing repository (`refs/git-bug/keylog`). The dedicated repo is cleaner for cross-repo sharing; the branch avoids requiring a second repository.

4. **Offline revocation.** Requiring network access for revocation is safe but harsh: a user whose device is stolen may be offline when they discover the theft. A potential mitigation is a grace-period revocation token — a pre-signed, time-limited capability generated at key-addition time and stored offline, redeemable without connectivity. This needs careful design to avoid introducing a new attack vector (the token itself becomes a credential that can be stolen).

5. **Community server discovery.** When a GitKeyLog backend is a git-bug community instance, how does a new participant discover and trust it? A per-repo `ProjectConfig` field naming the community identity server is one option; DNS-based discovery is another.
