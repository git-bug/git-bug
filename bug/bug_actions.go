package bug

import (
	"fmt"
	"io"
	"strings"

	"github.com/MichaelMure/git-bug/repository"
)

const MsgMergeNew = "new"
const MsgMergeInvalid = "invalid data"
const MsgMergeUpdated = "updated"
const MsgMergeNothing = "nothing to do"

func Fetch(repo repository.Repo, remote string) (string, error) {
	remoteRefSpec := fmt.Sprintf(bugsRemoteRefPattern, remote)
	fetchRefSpec := fmt.Sprintf("%s*:%s*", bugsRefPattern, remoteRefSpec)

	return repo.FetchRefs(remote, fetchRefSpec)
}

func Push(repo repository.Repo, remote string) (string, error) {
	return repo.PushRefs(remote, bugsRefPattern+"*")
}

func Pull(repo repository.Repo, out io.Writer, remote string) error {
	fmt.Fprintf(out, "Fetching remote ...\n")

	stdout, err := Fetch(repo, remote)
	if err != nil {
		return err
	}

	out.Write([]byte(stdout))

	fmt.Fprintf(out, "Merging data ...\n")

	for merge := range MergeAll(repo, remote) {
		if merge.Err != nil {
			return merge.Err
		}

		if merge.Status != MsgMergeNothing {
			fmt.Fprintf(out, "%s: %s\n", merge.HumanId, merge.Status)
		}
	}

	return nil
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
				out <- newMergeStatus(id, MsgMergeInvalid)
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

				out <- newMergeStatus(id, MsgMergeNew)
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
				out <- newMergeStatus(id, MsgMergeUpdated)
			} else {
				out <- newMergeStatus(id, MsgMergeNothing)
			}
		}
	}()

	return out
}
