package bug

import (
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"strings"
)

const MsgNew = "new"
const MsgInvalid = "invalid data"
const MsgUpdated = "updated"
const MsgNothing = "nothing to do"

func Fetch(repo repository.Repo, remote string) error {
	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", bugsRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

func Push(repo repository.Repo, remote string) error {
	return repo.PushRefs(remote, bugsRefPattern+"*")
}

type MergeResult struct {
	Err error

	Id      string
	HumanId string
	Status  string
}

func newMergeError(id string, err error) MergeResult {
	return MergeResult{
		Id:      id,
		HumanId: formatHumanId(id),
		Status:  err.Error(),
	}
}

func newMergeStatus(id string, status string) MergeResult {
	return MergeResult{
		Id:      id,
		HumanId: formatHumanId(id),
		Status:  status,
	}
}

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
				out <- newMergeError(id, err)
				continue
			}

			// Check for error in remote data
			if !remoteBug.IsValid() {
				out <- newMergeStatus(id, MsgInvalid)
				continue
			}

			localRef := bugsRefPattern + remoteBug.Id()
			localExist, err := repo.RefExist(localRef)

			// the bug is not local yet, simply create the reference
			if !localExist {
				err := repo.CopyRef(remoteRef, localRef)

				if err != nil {
					out <- newMergeError(id, err)
					return
				}

				out <- newMergeStatus(id, MsgNew)
				continue
			}

			localBug, err := readBug(repo, localRef)

			if err != nil {
				out <- newMergeError(id, err)
				return
			}

			updated, err := localBug.Merge(repo, remoteBug)

			if err != nil {
				out <- newMergeError(id, err)
				return
			}

			if updated {
				out <- newMergeStatus(id, MsgUpdated)
			} else {
				out <- newMergeStatus(id, MsgNothing)
			}
		}
	}()

	return out
}
