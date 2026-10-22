package identity

import (
	"context"
	"errors"
	"fmt"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// Fetch retrieve updates from a remote
// This does not change the local identities state
func Fetch(ctx context.Context, repo repository.Repo, remote string) (string, error) {
	return repo.FetchRefs(ctx, remote, Namespace)
}

// Push update a remote with the local changes
func Push(ctx context.Context, repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(ctx, remote, Namespace)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func Pull(ctx context.Context, repo repository.Repo, remote string) error {
	_, err := Fetch(ctx, repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(ctx, repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return fmt.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

// MergeAll will merge all the available remote identity
func MergeAll(ctx context.Context, repo repository.Repo, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefs, err := repo.ListTrackingRefs(remote, Namespace)

		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}

		for key, remoteCommit := range remoteRefs {
			if err := ctx.Err(); err != nil {
				out <- entity.NewMergeError(err, "")
				return
			}

			id := entity.Id(key)
			remoteIdentity, err := read(ctx, repo, id, remoteCommit)

			if err != nil {
				// a read cut short says nothing of the remote data
				if ctx.Err() != nil {
					out <- entity.NewMergeError(err, id)
					return
				}
				out <- entity.NewMergeInvalidStatus(id, fmt.Errorf("remote identity is not readable: %w", err).Error())
				continue
			}

			// Check for error in remote data
			if err := remoteIdentity.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, fmt.Errorf("remote identity is invalid: %w", err).Error())
				continue
			}

			localCommit, err := repo.ResolveRef(Namespace, id.String())

			// the identity is not local yet, simply create it
			if errors.Is(err, repository.ErrNotFound) {
				err := repo.UpdateRef(Namespace, id.String(), "", remoteCommit)

				if err != nil {
					out <- entity.NewMergeError(err, id)
					return
				}

				out <- entity.NewMergeNewStatus(id)
				continue
			}

			if err != nil {
				out <- entity.NewMergeError(err, id)
				continue
			}

			localIdentity, err := read(ctx, repo, id, localCommit)

			if err != nil {
				out <- entity.NewMergeError(fmt.Errorf("local identity is not readable: %w", err), id)
				return
			}

			updated, err := localIdentity.Merge(repo, remoteIdentity)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, fmt.Errorf("merge failed: %w", err).Error())
				return
			}

			if updated {
				out <- entity.NewMergeUpdatedStatus(id)
			} else {
				out <- entity.NewMergeNothingStatus(id)
			}
		}
	}()

	return out
}

// Remove will remove a local identity from its entity.Id.
// It is left as a responsibility to the caller to make sure that this identities is not
// linked from another entity, otherwise it would break it.
// Remove is idempotent.
func Remove(repo repository.Repo, id entity.Id) error {
	// list the remotes before deleting anything, to not stop halfway on failure
	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	err = repo.RemoveRef(Namespace, id.String())
	if err != nil {
		return err
	}

	for remote := range remotes {
		err = repo.RemoveTrackingRef(remote, Namespace, id.String())
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAll will remove all local identities.
// It is left as a responsibility to the caller to make sure that those identities are not
// linked from another entity, otherwise it would break them.
// RemoveAll is idempotent.
func RemoveAll(repo repository.Repo) error {
	localIds, err := ListLocalIds(repo)
	if err != nil {
		return err
	}
	for _, id := range localIds {
		err = Remove(repo, id)
		if err != nil {
			return err
		}
	}
	return nil
}
