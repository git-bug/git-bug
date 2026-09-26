package dag

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// ListLocalIds list all the available local Entity's Id
func ListLocalIds(def Definition, repo repository.RepoData) ([]entity.Id, error) {
	refs, err := repo.ListRefs(def.Namespace)
	if err != nil {
		return nil, err
	}
	ids := make([]entity.Id, 0, len(refs))
	for key := range refs {
		id := entity.Id(key)
		if err := id.Validate(); err != nil {
			return nil, errors.Wrapf(err, "invalid id %q", key)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Fetch retrieve updates from a remote
// This does not change the local entity state
func Fetch(def Definition, repo repository.Repo, remote string) (string, error) {
	return repo.FetchRefs(remote, def.Namespace)
}

// Push update a remote with the local changes
func Push(def Definition, repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, def.Namespace)
}

// Pull will do a Fetch + MergeAll
// Contrary to MergeAll, this function will return an error if a merge fail.
func Pull[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, author identity.Interface) error {
	_, err := Fetch(def, repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(def, wrapper, repo, resolvers, remote, author) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
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
//  5. if both local and remote Entity have new commits (that is, we have a concurrent edition),
//     a merge commit with an empty operationPack is created to join both branch and form a DAG.
//     --> emit entity.MergeStatusUpdated
//
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available. The author can be nil, in which
// case entities that require a merge commit will yield a merge error.
func MergeAll[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, remote string, author identity.Interface) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefs, err := repo.ListTrackingRefs(remote, def.Namespace)
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}

		for key, remoteCommit := range remoteRefs {
			out <- merge[EntityT](def, wrapper, repo, resolvers, entity.Id(key), remoteCommit, author)
		}
	}()

	return out
}

// merge perform a merge to make sure a local Entity is up-to-date.
// See MergeAll for more details.
func merge[EntityT entity.Interface](def Definition, wrapper func(e *Entity) EntityT, repo repository.ClockedRepo, resolvers entity.Resolvers, id entity.Id, remoteCommit repository.Hash, author identity.Interface) entity.MergeResult {
	if err := id.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid id").Error())
	}

	localCommit, err := repo.ResolveRef(def.Namespace, id.String())
	localExist := err == nil
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return entity.NewMergeError(err, id)
	}

	// Scenarios 1 and 2 don't change anything, so they are checked before reading
	// the remote Entity as there is no need to pay for it. Scenario 1 is by far
	// the most common case.

	if localExist {
		// SCENARIO 1
		// if the remote and local Entity have the same state, nothing is changed

		if localCommit == remoteCommit {
			// nothing to merge
			return entity.NewMergeNothingStatus(id)
		}

		// SCENARIO 2
		// if the local Entity has new commits but the remote don't, nothing is changed

		localCommits, err := repo.ListCommits(localCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}

		if slices.Contains(localCommits, remoteCommit) {
			return entity.NewMergeNothingStatus(id)
		}
	}

	remoteEntity, err := read[EntityT](def, wrapper, repo, resolvers, id, remoteCommit)
	if err != nil {
		return entity.NewMergeInvalidStatus(id,
			errors.Wrapf(err, "remote %s is not readable", def.Typename).Error())
	}

	// Check for error in remote data
	if err := remoteEntity.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id,
			errors.Wrapf(err, "remote %s data is invalid", def.Typename).Error())
	}

	// SCENARIO 3
	// if the remote Entity doesn't exist locally, it's created

	if !localExist {
		// the bug is not local yet, simply create it
		err := repo.UpdateRef(def.Namespace, id.String(), "", remoteCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}

		return entity.NewMergeNewStatus(id, remoteEntity)
	}

	// SCENARIO 4
	// if the remote has new commit, the local bug is updated to match the same history
	// (fast-forward update)

	remoteCommits, err := repo.ListCommits(remoteCommit)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// fast-forward is possible if otherRef include ref
	fastForwardPossible := slices.Contains(remoteCommits, localCommit)

	if fastForwardPossible {
		err = repo.UpdateRef(def.Namespace, id.String(), localCommit, remoteCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}
		return entity.NewMergeUpdatedStatus(id, remoteEntity)
	}

	// SCENARIO 5
	// if both local and remote Entity have new commits (that is, we have a concurrent edition),
	// a merge commit with an empty operationPack is created to join both branch and form a DAG.

	// fast-forward is not possible, we need to create a merge commit
	// For simplicity when reading and to have clocks that record this change, we store
	// an empty operationPack.
	// First step is to collect those clocks.

	// A merge commit needs an author. It's only required here, so that pulling
	// without a user identity set (e.g. in a fresh clone) works in all other cases.
	if author == nil {
		return entity.NewMergeError(errors.Wrapf(identity.ErrNoIdentitySet,
			"%s %s has diverged and requires a merge commit", def.Typename, id.Human()), id)
	}

	editTime, err := incrementClock(def, repo, editClockPattern)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	opp := &operationPack{
		Author:     author,
		Operations: nil,
		CreateTime: 0,
		EditTime:   editTime,
	}

	commitHash, err := opp.Write(def, repo, localCommit, remoteCommit)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// finally update the ref
	err = repo.UpdateRef(def.Namespace, id.String(), localCommit, commitHash)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// read the merged entity back, so that the returned entity holds the operations
	// of both branches and can be committed on top of the merge commit.
	mergedEntity, err := read[EntityT](def, wrapper, repo, resolvers, id, commitHash)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	return entity.NewMergeUpdatedStatus(id, mergedEntity)
}

// Remove delete an Entity, as well as its tracking refs for every remote.
// Remove is idempotent.
func Remove(def Definition, repo repository.ClockedRepo, id entity.Id) error {
	// list the remotes before deleting anything, to not stop halfway on failure
	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	err = repo.RemoveRef(def.Namespace, id.String())
	if err != nil {
		return err
	}

	for remote := range remotes {
		err = repo.RemoveTrackingRef(remote, def.Namespace, id.String())
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAll delete all Entity matching the Definition.
// RemoveAll is idempotent.
func RemoveAll(def Definition, repo repository.ClockedRepo) error {
	localIds, err := ListLocalIds(def, repo)
	if err != nil {
		return err
	}
	for _, id := range localIds {
		err = Remove(def, repo, id)
		if err != nil {
			return err
		}
	}
	return nil
}
