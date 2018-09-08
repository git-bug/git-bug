package bug

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
)

const MsgMergeNew = "new"
const MsgMergeInvalid = "invalid data"
const MsgMergeUpdated = "updated"
const MsgMergeNothing = "nothing to do"

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

type MergeResult struct {
	// Err is set when a terminal error occur in the process
	Err error

	Id     string
	Status string
	Bug    *Bug
}

func newMergeError(err error, id string) MergeResult {
	return MergeResult{
		Err: err,
		Id:  id,
	}
}

func newMergeStatus(status string, id string, bug *Bug) MergeResult {
	return MergeResult{
		Id:     id,
		Status: status,

		// Bug is not set for an invalid merge result
		Bug: bug,
	}
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
				out <- newMergeError(err, id)
				continue
			}

			// Check for error in remote data
			if !remoteBug.IsValid() {
				out <- newMergeStatus(MsgMergeInvalid, id, nil)
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

				out <- newMergeStatus(MsgMergeNew, id, remoteBug)
				continue
			}

			localBug, err := readBug(repo, localRef)

			if err != nil {
				out <- newMergeError(err, id)
				return
			}

			updated, err := localBug.Merge(repo, remoteBug)

			if err != nil {
				out <- newMergeError(err, id)
				return
			}

			if updated {
				out <- newMergeStatus(MsgMergeUpdated, id, localBug)
			} else {
				out <- newMergeStatus(MsgMergeNothing, id, localBug)
			}
		}
	}()

	return out
}
