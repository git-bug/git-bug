package bug

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/pkg/errors"
)

// Fetch retrieve updates from a remote
// This does not change the local bugs state
func Fetch(repo repository.Repo, remote string) (string, error) {
	// "refs/bugs/*:refs/remotes/<remote>>/bugs/*"
	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", bugsRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string) (string, error) {
	// "refs/bugs/*:refs/bugs/*"
	refspec := fmt.Sprintf("%s*:%s*", bugsRefPattern, bugsRefPattern)

	return repo.PushRefs(remote, refspec)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func Pull(repo repository.ClockedRepo, remote string) error {
	_, err := Fetch(repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

// MergeAll will merge all the available remote bug:
//
// - If the remote has new commit, the local bug is updated to match the same history
//   (fast-forward update)
// - if the local bug has new commits but the remote don't, nothing is changed
// - if both local and remote bug have new commits (that is, we have a concurrent edition),
//   new local commits are rewritten at the head of the remote history (that is, a rebase)
func MergeAll(repo repository.ClockedRepo, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// no caching for the merge, we load everything from git even if that means multiple
	// copy of the same entity in memory. The cache layer will intercept the results to
	// invalidate entities if necessary.
	identityResolver := identity.NewSimpleResolver(repo)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)
		if err != nil {
			out <- entity.MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			refSplit := strings.Split(remoteRef, "/")
			id := entity.Id(refSplit[len(refSplit)-1])

			if err := id.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "invalid ref").Error())
				continue
			}

			remoteBug, err := read(repo, identityResolver, remoteRef)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote bug is not readable").Error())
				continue
			}

			// Check for error in remote data
			if err := remoteBug.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote bug is invalid").Error())
				continue
			}

			localRef := bugsRefPattern + remoteBug.Id().String()
			localExist, err := repo.RefExist(localRef)

			if err != nil {
				out <- entity.NewMergeError(err, id)
				continue
			}

			// the bug is not local yet, simply create the reference
			if !localExist {
				err := repo.CopyRef(remoteRef, localRef)

				if err != nil {
					out <- entity.NewMergeError(err, id)
					return
				}

				out <- entity.NewMergeStatus(entity.MergeStatusNew, id, remoteBug)
				continue
			}

			localBug, err := read(repo, identityResolver, localRef)

			if err != nil {
				out <- entity.NewMergeError(errors.Wrap(err, "local bug is not readable"), id)
				return
			}

			updated, err := localBug.Merge(repo, remoteBug)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "merge failed").Error())
				return
			}

			if updated {
				out <- entity.NewMergeStatus(entity.MergeStatusUpdated, id, localBug)
			} else {
				out <- entity.NewMergeStatus(entity.MergeStatusNothing, id, localBug)
			}
		}
	}()

	return out
}
