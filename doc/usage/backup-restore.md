# Backup and restore

`git-bug` data lives entirely in git refs (`refs/bugs/*` and `refs/identities/*`),
so a plain `git bundle` of those refs is a complete, verifiable, restorable
backup. This is the recommended way to back up a store, or to roll back a store
that was mutated by a bridge round trip or an experiment.

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

## Cache invalidation

Deleting a bug ref has **no visible effect** on its own: `git bug` keeps listing
the deleted bug from `.git/git-bug/cache/bugs` until that file is removed. After
restoring, clear the cache so the deletion is observed:

```sh
rm .git/git-bug/cache/bugs
```
