package identity

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/pkg/errors"
)

// Fetch retrieve updates from a remote
// This does not change the local identities state
func Fetch(repo repository.Repo, remote string) (string, error) {
	remoteRefSpec := fmt.Sprintf(identityRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s:%s*", identityRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, identityRefPattern+"*")
}

// Pull will do a Fetch + MergeAll
// This function won't give details on the underlying process. If you need more,
// use Fetch and MergeAll separately.
func Pull(repo repository.ClockedRepo, remote string) error {
	_, err := Fetch(repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == MergeStatusInvalid {
			// Not awesome: simply output the merge failure here as this function
			// is only used in tests for now.
			fmt.Println(merge)
		}
	}

	return nil
}

// MergeAll will merge all the available remote identity
// To make sure that an Identity history can't be altered, a strict fast-forward
// only policy is applied here. As an Identity should be tied to a single user, this
// should work in practice but it does leave a possibility that a user would edit his
// Identity from two different repo concurrently and push the changes in a non-centralized
// network of repositories. In this case, it would result some of the repo accepting one
// version, some other accepting another, preventing the network in general to converge
// to the same result. This would create a sort of partition of the network, and manual
// cleaning would be required.
//
// An alternative approach would be to have a determinist rebase:
// - any commits present in both local and remote version would be kept, never changed.
// - newer commits would be merged in a linear chain of commits, ordered based on the
//   Lamport time
//
// However, this approach leave the possibility, in the case of a compromised crypto keys,
// of forging a new version with a bogus Lamport time to be inserted before a legit version,
// invalidating the correct version and hijacking the Identity. There would only be a short
// period of time where this would be possible (before the network converge) but I'm not
// confident enough to implement that. I choose the strict fast-forward only approach,
// despite it's potential problem with two different version as mentioned above.
func MergeAll(repo repository.ClockedRepo, remote string) <-chan MergeResult {
	out := make(chan MergeResult)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf(identityRemoteRefPattern, remote)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)

		if err != nil {
			out <- MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			refSplitted := strings.Split(remoteRef, "/")
			id := refSplitted[len(refSplitted)-1]

			remoteIdentity, err := ReadLocal(repo, remoteRef)
			remoteBug, err := readBug(repo, remoteRef)

			if err != nil {
				out <- newMergeInvalidStatus(id, errors.Wrap(err, "remote bug is not readable").Error())
				continue
			}

			// Check for error in remote data
			if err := remoteBug.Validate(); err != nil {
				out <- newMergeInvalidStatus(id, errors.Wrap(err, "remote bug is invalid").Error())
				continue
			}

			localRef := bugsRefPattern + remoteBug.Id()
			localExist, err := repo.RefExist(localRef)

			if err != nil {
				out <- newMergeError(err, id)
				continue
			}

			// the bug is not local yet, simply create the reference
			if !localExist {
				err := repo.CopyRef(remoteRef, localRef)

				if err != nil {
					out <- newMergeError(err, id)
					return
				}

				out <- newMergeStatus(MergeStatusNew, id, remoteBug)
				continue
			}

			localBug, err := readBug(repo, localRef)

			if err != nil {
				out <- newMergeError(errors.Wrap(err, "local bug is not readable"), id)
				return
			}

			updated, err := localBug.Merge(repo, remoteBug)

			if err != nil {
				out <- newMergeInvalidStatus(id, errors.Wrap(err, "merge failed").Error())
				return
			}

			if updated {
				out <- newMergeStatus(MergeStatusUpdated, id, localBug)
			} else {
				out <- newMergeStatus(MergeStatusNothing, id, localBug)
			}
		}
	}()

	return out
}

// MergeStatus represent the result of a merge operation of a bug
type MergeStatus int

const (
	_ MergeStatus = iota
	MergeStatusNew
	MergeStatusInvalid
	MergeStatusUpdated
	MergeStatusNothing
)

// Todo: share a generalized MergeResult with the bug package ?
type MergeResult struct {
	// Err is set when a terminal error occur in the process
	Err error

	Id     string
	Status MergeStatus

	// Only set for invalid status
	Reason string

	// Not set for invalid status
	Identity *Identity
}

func (mr MergeResult) String() string {
	switch mr.Status {
	case MergeStatusNew:
		return "new"
	case MergeStatusInvalid:
		return fmt.Sprintf("invalid data: %s", mr.Reason)
	case MergeStatusUpdated:
		return "updated"
	case MergeStatusNothing:
		return "nothing to do"
	default:
		panic("unknown merge status")
	}
}

func newMergeError(err error, id string) MergeResult {
	return MergeResult{
		Err: err,
		Id:  id,
	}
}

func newMergeStatus(status MergeStatus, id string, identity *Identity) MergeResult {
	return MergeResult{
		Id:     id,
		Status: status,

		// Identity is not set for an invalid merge result
		Identity: identity,
	}
}

func newMergeInvalidStatus(id string, reason string) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusInvalid,
		Reason: reason,
	}
}
