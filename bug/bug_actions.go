package bug

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/pkg/errors"
)

// Fetch retrieve update from a remote
// This does not change the local bugs state
func Fetch(repo repository.Repo, remote string) (string, error) {
	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", bugsRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

// Push update a remote with the local changes
func Push(repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, bugsRefPattern+"*")
}

// Pull will do a Fetch + MergeAll
// This function won't give details on the underlying process. If you need more
// use Fetch and MergeAll separately.
func Pull(repo repository.Repo, remote string) error {
	_, err := Fetch(repo, remote)
	if err != nil {
		return err
	}

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}
	}

	return nil
}

// MergeAll will merge all the available remote bug
func MergeAll(repo repository.Repo, remote string) <-chan MergeResult {
	out := make(chan MergeResult)

	go func() {
		defer close(out)

		remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
		remoteRefs, err := repo.ListRefs(remoteRefSpec)

		if err != nil {
			out <- MergeResult{Err: err}
			return
		}

		for _, remoteRef := range remoteRefs {
			refSplitted := strings.Split(remoteRef, "/")
			id := refSplitted[len(refSplitted)-1]

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

type MergeResult struct {
	// Err is set when a terminal error occur in the process
	Err error

	Id     string
	Status MergeStatus

	// Only set for invalid status
	Reason string

	// Not set for invalid status
	Bug *Bug
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

func newMergeStatus(status MergeStatus, id string, bug *Bug) MergeResult {
	return MergeResult{
		Id:     id,
		Status: status,

		// Bug is not set for an invalid merge result
		Bug: bug,
	}
}

func newMergeInvalidStatus(id string, reason string) MergeResult {
	return MergeResult{
		Id:     id,
		Status: MergeStatusInvalid,
		Reason: reason,
	}
}
