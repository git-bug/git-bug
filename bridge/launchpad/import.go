package launchpad

import (
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/pkg/errors"
)

type launchpadImporter struct {
	conf core.Configuration
}

func (li *launchpadImporter) Init(conf core.Configuration) error {
	li.conf = conf
	return nil
}

const keyLaunchpadID = "launchpad-id"

func (li *launchpadImporter) makePerson(owner LPPerson) bug.Person {
	return bug.Person{
		Name:      owner.Name,
		Email:     "",
		Login:     owner.Login,
		AvatarUrl: "",
	}
}

func (li *launchpadImporter) ImportAll(repo *cache.RepoCache) error {
	lpAPI := new(launchpadAPI)

	err := lpAPI.Init()
	if err != nil {
		return err
	}

	lpBugs, err := lpAPI.SearchTasks(li.conf["project"])
	if err != nil {
		return err
	}

	for _, lpBug := range lpBugs {
		var b *cache.BugCache
		var err error

		lpBugID := fmt.Sprintf("%d", lpBug.ID)
		b, err = repo.ResolveBugCreateMetadata(keyLaunchpadID, lpBugID)
		if err != nil && err != bug.ErrBugNotExist {
			return err
		}

		if err == bug.ErrBugNotExist {
			createdAt, _ := time.Parse(time.RFC3339, lpBug.CreatedAt)
			b, err = repo.NewBugRaw(
				li.makePerson(lpBug.Owner),
				createdAt.Unix(),
				lpBug.Title,
				lpBug.Description,
				nil,
				map[string]string{
					keyLaunchpadID: lpBugID,
				},
			)
			if err != nil {
				return errors.Wrapf(err, "failed to add bug id #%s", lpBugID)
			}
		} else {
			/* TODO: Update bug */
			fmt.Println("TODO: Update bug")
		}

		/* Handle messages */
		if len(lpBug.Messages) == 0 {
			return errors.Wrapf(err, "failed to fetch comments for bug #%s", lpBugID)
		}

		// The Launchpad API returns the bug description as the first
		// comment, so skip it.
		for _, lpMessage := range lpBug.Messages[1:] {
			_, err := b.ResolveTargetWithMetadata(keyLaunchpadID, lpMessage.ID)
			if err != nil && err != cache.ErrNoMatchingOp {
				return errors.Wrapf(err, "failed to fetch comments for bug #%s", lpBugID)
			}

			// If this comment already exists, we are probably
			// updating an existing bug. We do not want to duplicate
			// the comments, so let us just skip this one.
			// TODO: Can Launchpad comments be edited?
			if err == nil {
				continue
			}

			// This is a new comment, we can add it.
			createdAt, _ := time.Parse(time.RFC3339, lpMessage.CreatedAt)
			err = b.AddCommentRaw(
				li.makePerson(lpMessage.Owner),
				createdAt.Unix(),
				lpMessage.Content,
				nil,
				map[string]string{
					keyLaunchpadID: lpMessage.ID,
				})
			if err != nil {
				return errors.Wrapf(err, "failed to add comment to bug #%s", lpBugID)
			}
		}
		err = b.CommitAsNeeded()
		if err != nil {
			return err
		}
	}
	return nil
}

func (li *launchpadImporter) Import(repo *cache.RepoCache, id string) error {
	fmt.Println("IMPORT")
	return nil
}
