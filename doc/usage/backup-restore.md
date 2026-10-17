# Backup and restore

`git-bug` data lives entirely in git refs (`refs/bugs/*` and `refs/identities/*`),
so a plain `git bundle` of those refs is a complete, verifiable, restorable
backup of a store. This is the recommended way to back up, or to roll back a store
that was mutated by a bridge round trip or an experiment.

This backs up issues and identities, not secrets: the private keys used to sign
operations, and the credentials used by bridges, live in the keyring, outside the
repository, under `git-bug/keyring` in the configuration directory of the user.
Copy that directory too if the machine itself may be lost, or a restored store
holds issues that can no longer be signed as their authors.

`git bug --format json` is **not** a restore path: it is a lossy read-model
snapshot, with no way to reconstruct the underlying operation DAG from it.

## Backup

```sh
git for-each-ref --format='%(refname)' | grep -E 'refs/(bugs|identities)' > refs.txt
git bundle create git-bug-refs.bundle $(cat refs.txt | tr '\n' ' ')
git bundle verify git-bug-refs.bundle   # -> "The bundle records a complete history"
```

## Restore

Resets modified bugs to the backed-up state:

```sh
git fetch --force git-bug-refs.bundle 'refs/bugs/*:refs/bugs/*' 'refs/identities/*:refs/identities/*'
```

Drop bugs created after the backup (not present in the bundle, so unaffected by
`--force`):

```sh
git update-ref -d refs/bugs/<full-hash>
```

Every command reconciles its index against the references before answering, so a
deleted or restored reference is observed by the next `git bug` call: no manual
cache invalidation is needed.
