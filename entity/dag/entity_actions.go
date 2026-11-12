package dag

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// ListLocalIds list all the available local Entity's Id.
// A singleton is not addressed by id, see ReadSingleton.
func ListLocalIds(ctx context.Context, def Definition, repo repository.RepoData) ([]entity.Id, error) {
	if def.isSingleton() {
		return nil, fmt.Errorf("%s is a singleton, see ReadSingleton", def.Typename)
	}
	heads, err := def.listEntities(ctx, repo)
	if err != nil {
		return nil, err
	}
	return slices.Collect(maps.Keys(heads)), nil
}

// ListHeads returns every local Entity's Id, with the commit its reference points to.
// A singleton is not addressed by id, see ReadSingleton.
func ListHeads(ctx context.Context, def Definition, repo repository.Repo) (map[entity.Id]repository.Hash, error) {
	if def.isSingleton() {
		return nil, fmt.Errorf("%s is a singleton, see ReadSingleton", def.Typename)
	}
	return def.listEntities(ctx, repo)
}

// Head returns the commit the reference of a local Entity points to, or
// repository.ErrNotFound if there is none.
// A singleton is not addressed by id, see ReadSingleton.
func Head(ctx context.Context, def Definition, repo repository.Repo, id entity.Id) (repository.Hash, error) {
	if def.isSingleton() {
		return "", fmt.Errorf("%s is a singleton, see ReadSingleton", def.Typename)
	}
	return repo.ResolveRef(ctx, def.Namespace, def.keyFromId(id))
}

// IdsFromKeys returns the ids of the Entities stored in the references with these keys,
// ignoring the other keys, including those of the outdated repository format which
// ListHeads reports. It is empty for a singleton, which is not addressed by id.
func IdsFromKeys(def Definition, keys []string) []entity.Id {
	if def.isSingleton() {
		return nil
	}
	ids := make([]entity.Id, 0, len(keys))
	for _, key := range keys {
		if id, ok, _ := def.idFromKey(key); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// Fetch retrieve updates from a remote
// This does not change the local entity state
func Fetch(ctx context.Context, def Definition, repo repository.Repo, remote string) (string, error) {
	return repo.FetchRefs(ctx, remote, def.Namespace)
}

// Push update a remote with the local changes
func Push(ctx context.Context, def Definition, repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(ctx, remote, def.Namespace)
}

// Pull will do a Fetch + MergeAll
// Contrary to MergeAll, this function will return an error if a merge fail.
func Pull[EntityT entity.Interface](ctx context.Context, def Definition, wrapper func(e *Entity) EntityT, repo repository.Repo, resolvers entity.Resolvers, remote string, author identity.Interface) error {
	_, err := Fetch(ctx, def, repo, remote)
	if err != nil {
		return err
	}

	// MergeAll needs to be read to the end, so the first failure is kept for later
	var firstErr error
	for merge := range MergeAll(ctx, def, wrapper, repo, resolvers, remote, author) {
		if firstErr != nil {
			continue
		}
		if merge.Err != nil {
			firstErr = merge.Err
		} else if merge.Status == entity.MergeStatusInvalid || merge.Status == entity.MergeStatusConflict {
			firstErr = fmt.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return firstErr
}

// MergeAll will merge all the available remote Entity:
//
// Multiple scenario exist:
//  1. if the remote and local Entity have the same state, nothing is changed
//     --> emit entity.MergeStatusNothing
//  2. if the local Entity has new commits but the remote don't, nothing is changed
//     --> emit entity.MergeStatusNothing
//  3. if the remote Entity doesn't exist locally, it's created
//     --> emit entity.MergeStatusNew
//  4. if the remote has new commit, the local bug is updated to match the same history
//     (fast-forward update)
//     --> emit entity.MergeStatusUpdated
//  5. if the local and remote Entity have no common history (different root commits), they are
//     never merged, nothing is changed. For a singleton, they are different Entities under the
//     same reference.
//     --> emit entity.MergeStatusConflict
//  6. if both local and remote Entity have new commits (that is, we have a concurrent edition),
//     a merge commit with an empty operationPack is created to join both branch and form a DAG.
//     --> emit entity.MergeStatusUpdated
//
// A remote reference whose key can't hold an Entity is ignored, and nothing is emitted for it.
// The key of an Entity in the outdated repository format emits entity.MergeStatusInvalid.
//
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available. The author can be nil, in which
// case entities that require a merge commit will yield a merge error.
func MergeAll[EntityT entity.Interface](ctx context.Context, def Definition, wrapper func(e *Entity) EntityT, repo repository.Repo, resolvers entity.Resolvers, remote string, author identity.Interface) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefs, err := repo.ListTrackingRefs(ctx, remote, def.Namespace)
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}

		for key, remoteCommit := range remoteRefs {
			if err := ctx.Err(); err != nil {
				out <- entity.NewMergeError(err, "")
				return
			}
			// the id of a regular Entity, empty for a singleton which is not addressed by id
			id, ok, err := def.idFromKey(key)
			if err != nil {
				out <- entity.NewMergeInvalidStatus(entity.Id(key), fmt.Errorf("invalid id: %w", err).Error())
				continue
			}
			if !ok {
				continue
			}
			result := merge[EntityT](ctx, def, wrapper, repo, resolvers, key, id, remoteCommit, author)
			out <- result
			if result.Err != nil && ctx.Err() != nil {
				// cut short, already reported
				return
			}
		}
	}()

	return out
}

// merge perform a merge to make sure the local Entity of the reference with this key is
// up-to-date. id is the id the Entity must have, see Definition.idFromKey.
// See MergeAll for more details.
func merge[EntityT entity.Interface](ctx context.Context, def Definition, wrapper func(e *Entity) EntityT, repo repository.Repo, resolvers entity.Resolvers, key string, id entity.Id, remoteCommit repository.Hash, author identity.Interface) entity.MergeResult {
	localCommit, err := repo.ResolveRef(ctx, def.Namespace, key)
	localExist := err == nil
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return entity.NewMergeError(err, id)
	}

	// Scenarios 1 and 2 don't change anything, so they are checked before reading
	// the remote Entity as there is no need to pay for it. Scenario 1 is by far
	// the most common case.

	var localRoot repository.Hash
	if localExist {
		// SCENARIO 1
		// if the remote and local Entity have the same state, nothing is changed

		if localCommit == remoteCommit {
			// nothing to merge
			return entity.NewMergeNothingStatus(id)
		}

		// SCENARIO 2
		// if the local Entity has new commits but the remote don't, nothing is changed

		var localCommits map[repository.Hash]repository.Commit
		localCommits, localRoot, err = walkHistory(ctx, repo, localCommit, "")
		if err != nil {
			return entity.NewMergeError(err, id)
		}

		if _, ok := localCommits[remoteCommit]; ok {
			return entity.NewMergeNothingStatus(id)
		}
	}

	remoteEntity, remoteCommits, err := read[EntityT](ctx, def, wrapper, repo, resolvers, id, remoteCommit)
	if err != nil {
		// a read cut short says nothing of the remote data
		if ctx.Err() != nil {
			return entity.NewMergeError(err, id)
		}
		return entity.NewMergeInvalidStatus(id, fmt.Errorf("remote %s is not readable: %w", def.Typename, err).Error())
	}

	// Check for error in remote data
	if err := remoteEntity.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id, fmt.Errorf("remote %s data is invalid: %w", def.Typename, err).Error())
	}

	// SCENARIO 3
	// if the remote Entity doesn't exist locally, it's created

	if !localExist {
		// the bug is not local yet, simply create it
		err := repo.UpdateRef(ctx, def.Namespace, key, "", remoteCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}

		return entity.NewMergeNewStatus(id)
	}

	// SCENARIO 4
	// if the remote has new commit, the local bug is updated to match the same history
	// (fast-forward update)

	// fast-forward is possible if the remote history includes the local commit
	if _, ok := remoteCommits[localCommit]; ok {
		err = repo.UpdateRef(ctx, def.Namespace, key, localCommit, remoteCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}
		return entity.NewMergeUpdatedStatus(id)
	}

	// SCENARIO 5
	// if the local and remote Entity have no common history, they are never merged:
	// joining them would give a DAG with two roots, which can't be read. Both histories
	// descend from their single root, so they share a commit if and only if they share
	// the root. This is expected for a singleton, as its reference key doesn't fix its
	// first operation.

	if _, ok := remoteCommits[localRoot]; !ok {
		return entity.NewMergeConflictStatus(id, fmt.Sprintf("remote %s has no common history with the local one", def.entityName(id)))
	}

	// SCENARIO 6
	// if both local and remote Entity have new commits (that is, we have a concurrent edition),
	// a merge commit with an empty operationPack is created to join both branch and form a DAG.

	// fast-forward is not possible, we need to create a merge commit
	// For simplicity when reading and to have clocks that record this change, we store
	// an empty operationPack.
	// First step is to collect those clocks.

	// A merge commit needs an author. It's only required here, so that pulling
	// without a user identity set (e.g. in a fresh clone) works in all other cases.
	if author == nil {
		return entity.NewMergeError(fmt.Errorf(
			"%s has diverged and requires a merge commit: %w", def.entityName(id), identity.ErrNoIdentitySet,
		), id)
	}

	editTime, err := incrementClock(ctx, def, repo, def.editClock())
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	opp := &operationPack{
		Author:     author,
		Operations: nil,
		CreateTime: 0,
		EditTime:   editTime,
	}

	commitHash, err := opp.Write(ctx, def, repo, localCommit, remoteCommit)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// finally update the ref
	err = repo.UpdateRef(ctx, def.Namespace, key, localCommit, commitHash)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	return entity.NewMergeUpdatedStatus(id)
}

// Remove delete an Entity, as well as its tracking refs for every remote.
// Remove is idempotent.
// A singleton is not addressed by id, see RemoveAll.
func Remove(ctx context.Context, def Definition, repo repository.Repo, id entity.Id) error {
	if def.isSingleton() {
		return fmt.Errorf("%s is a singleton, see RemoveAll", def.Typename)
	}
	if err := id.Validate(); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return removeRef(ctx, def, repo, def.keyFromId(id))
}

// removeRef deletes the reference with this key, as well as its tracking refs for
// every remote.
func removeRef(ctx context.Context, def Definition, repo repository.Repo, key string) error {
	// list the remotes before deleting anything, to not stop halfway on failure
	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	err = repo.RemoveRef(ctx, def.Namespace, key)
	if err != nil {
		return err
	}

	// once the local ref is gone, RemoveAll can't find the entity anymore: its
	// tracking refs must be removed now, cancelled or not
	cleanupCtx := context.WithoutCancel(ctx)
	for remote := range remotes {
		err = repo.RemoveTrackingRef(cleanupCtx, remote, def.Namespace, key)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAll delete all Entity matching the Definition.
// RemoveAll is idempotent.
func RemoveAll(ctx context.Context, def Definition, repo repository.Repo) error {
	// an outdated reference fails the listing, before anything is removed
	heads, err := def.listEntities(ctx, repo)
	if err != nil {
		return err
	}
	for id := range heads {
		err = removeRef(ctx, def, repo, def.keyFromId(id))
		if err != nil {
			return err
		}
	}
	return nil
}
