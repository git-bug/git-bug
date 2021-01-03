package dag

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

func ListLocalIds(typename string, repo repository.RepoData) ([]entity.Id, error) {
	refs, err := repo.ListRefs(fmt.Sprintf("refs/%s/", typename))
	if err != nil {
		return nil, err
	}
	return entity.RefsToIds(refs), nil
}

// Fetch retrieve updates from a remote
// This does not change the local entity state
func Fetch(def Definition, repo repository.Repo, remote string) (string, error) {
	// "refs/<entity>/*:refs/remotes/<remote>/<entity>/*"
	fetchRefSpec := fmt.Sprintf("refs/%s/*:refs/remotes/%s/%s/*",
		def.namespace, remote, def.namespace)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with the local changes
func Push(def Definition, repo repository.Repo, remote string) (string, error) {
	// "refs/<entity>/*:refs/<entity>/*"
	refspec := fmt.Sprintf("refs/%s/*:refs/%s/*",
		def.namespace, def.namespace)

	return repo.PushRefs(remote, refspec)
}

// Pull will do a Fetch + MergeAll
// Contrary to MergeAll, this function will return an error if a merge fail.
func Pull(def Definition, repo repository.ClockedRepo, remote string) error {
	_, err := Fetch(def, repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(def, repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

func MergeAll(def Definition, repo repository.ClockedRepo, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// no caching for the merge, we load everything from git even if that means multiple
	// copy of the same entity in memory. The cache layer will intercept the results to
	// invalidate entities if necessary.

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf("refs/remotes/%s/%s/", remote, def.namespace)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)
		if err != nil {
			out <- entity.MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			out <- merge(def, repo, remoteRef)
		}
	}()

	return out
}

func merge(def Definition, repo repository.ClockedRepo, remoteRef string) entity.MergeResult {
	id := entity.RefToId(remoteRef)

	if err := id.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid ref").Error())
	}

	remoteEntity, err := read(def, repo, remoteRef)
	if err != nil {
		return entity.NewMergeInvalidStatus(id,
			errors.Wrapf(err, "remote %s is not readable", def.typename).Error())
	}

	// Check for error in remote data
	if err := remoteEntity.Validate(); err != nil {
		return entity.NewMergeInvalidStatus(id,
			errors.Wrapf(err, "remote %s data is invalid", def.typename).Error())
	}

	localRef := fmt.Sprintf("refs/%s/%s", def.namespace, id.String())

	localExist, err := repo.RefExist(localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// the bug is not local yet, simply create the reference
	if !localExist {
		err := repo.CopyRef(remoteRef, localRef)
		if err != nil {
			return entity.NewMergeError(err, id)
		}

		return entity.NewMergeStatus(entity.MergeStatusNew, id, remoteEntity)
	}

	// var updated bool
	// err = repo.MergeRef(localRef, remoteRef, func() repository.Hash {
	// 	updated = true
	//
	// })
	// if err != nil {
	// 	return entity.NewMergeError(err, id)
	// }
	//
	// if updated {
	// 	return entity.NewMergeStatus(entity.MergeStatusUpdated, id, )
	// } else {
	// 	return entity.NewMergeStatus(entity.MergeStatusNothing, id, )
	// }

	localCommit, err := repo.ResolveRef(localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	remoteCommit, err := repo.ResolveRef(remoteRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	if localCommit == remoteCommit {
		// nothing to merge
		return entity.NewMergeStatus(entity.MergeStatusNothing, id, remoteEntity)
	}

	// fast-forward is possible if otherRef include ref

	remoteCommits, err := repo.ListCommits(remoteRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	fastForwardPossible := false
	for _, hash := range remoteCommits {
		if hash == localCommit {
			fastForwardPossible = true
			break
		}
	}

	if fastForwardPossible {
		err = repo.UpdateRef(localRef, remoteCommit)
		if err != nil {
			return entity.NewMergeError(err, id)
		}
		return entity.NewMergeStatus(entity.MergeStatusUpdated, id, remoteEntity)
	}

	// fast-forward is not possible, we need to create a merge commit
	// For simplicity when reading and to have clocks that record this change, we store
	// an empty operationPack.
	// First step is to collect those clocks.

	localEntity, err := read(def, repo, localRef)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// err = localEntity.packClock.Witness(remoteEntity.packClock.Time())
	// if err != nil {
	// 	return entity.NewMergeError(err, id)
	// }
	//
	// packTime, err := localEntity.packClock.Increment()
	// if err != nil {
	// 	return entity.NewMergeError(err, id)
	// }

	editTime, err := repo.Increment(fmt.Sprintf(editClockPattern, def.namespace))
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	opp := &operationPack{
		Operations: nil,
		CreateTime: 0,
		EditTime:   editTime,
		// PackTime:   packTime,
	}

	treeHash, err := opp.Write(def, repo)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// Create the merge commit with two parents
	newHash, err := repo.StoreCommit(treeHash, localCommit, remoteCommit)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	// finally update the ref
	err = repo.UpdateRef(localRef, newHash)
	if err != nil {
		return entity.NewMergeError(err, id)
	}

	return entity.NewMergeStatus(entity.MergeStatusUpdated, id, localEntity)
}

func Remove() error {
	panic("")
}
