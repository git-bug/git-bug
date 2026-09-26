package identity

import (
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

// Fetch retrieve updates from a remote
// This does not change the local identities state
func Fetch(repo repository.Repo, remote string) (string, error) {
	return repo.FetchRefs(remote, Namespace)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, Namespace)
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

// MergeAll will merge all the available remote identity
func MergeAll(repo repository.ClockedRepo, remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	go func() {
		defer close(out)

		remoteRefs, err := repo.ListTrackingRefs(remote, Namespace)

		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}

		for key, remoteCommit := range remoteRefs {
			id := entity.Id(key)
			remoteIdentity, err := read(repo, id, remoteCommit)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote identity is not readable").Error())
				continue
			}

			// Check for error in remote data
			if err := remoteIdentity.Validate(); err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "remote identity is invalid").Error())
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

				out <- entity.NewMergeNewStatus(id, remoteIdentity)
				continue
			}

			if err != nil {
				out <- entity.NewMergeError(err, id)
				continue
			}

			localIdentity, err := read(repo, id, localCommit)

			if err != nil {
				out <- entity.NewMergeError(errors.Wrap(err, "local identity is not readable"), id)
				return
			}

			updated, err := localIdentity.Merge(repo, remoteIdentity)

			if err != nil {
				out <- entity.NewMergeInvalidStatus(id, errors.Wrap(err, "merge failed").Error())
				return
			}

			if updated {
				out <- entity.NewMergeUpdatedStatus(id, localIdentity)
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
func Remove(repo repository.ClockedRepo, id entity.Id) error {
	// list the remotes before deleting anything, to not stop halfway on failure
	remotes, err := repo.GetRemotes()
	if err != nil {
		return err
	}

	found, err := existAnywhere(repo, remotes, id)
	if err != nil {
		return err
	}
	if !found {
		return entity.NewErrNotFound(Typename)
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

// existAnywhere tells if an identity exists locally or in the tracking refs of
// any of the given remotes.
func existAnywhere(repo repository.ClockedRepo, remotes map[string]string, id entity.Id) (bool, error) {
	_, err := repo.ResolveRef(Namespace, id.String())
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return false, err
	}

	for remote := range remotes {
		_, err := repo.ResolveTrackingRef(remote, Namespace, id.String())
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return false, err
		}
	}

	return false, nil
}

// RemoveAll will remove all local identities.
// It is left as a responsibility to the caller to make sure that those identities are not
// linked from another entity, otherwise it would break them.
// RemoveAll is idempotent.
func RemoveAll(repo repository.ClockedRepo) error {
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
