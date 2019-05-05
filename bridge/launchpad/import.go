package launchpad

import (
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
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
const keyLaunchpadLogin = "launchpad-login"

func (li *launchpadImporter) ensurePerson(repo *cache.RepoCache, owner LPPerson) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(keyLaunchpadLogin, owner.Login)
	if err == nil {
		return i, nil
	}
	if _, ok := err.(identity.ErrMultipleMatch); ok {
		return nil, err
	}

	return repo.NewIdentityRaw(
		owner.Name,
		"",
		owner.Login,
		"",
		map[string]string{
			keyLaunchpadLogin: owner.Login,
		},
	)
}

func (li *launchpadImporter) ImportAll(repo *cache.RepoCache, since time.Time) error {
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

		owner, err := li.ensurePerson(repo, lpBug.Owner)
		if err != nil {
			return err
		}

		if err == bug.ErrBugNotExist {
			createdAt, _ := time.Parse(time.RFC3339, lpBug.CreatedAt)
			b, err = repo.NewBugRaw(
				owner,
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
			_, err := b.ResolveOperationWithMetadata(keyLaunchpadID, lpMessage.ID)
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

			owner, err := li.ensurePerson(repo, lpMessage.Owner)
			if err != nil {
				return err
			}

			// This is a new comment, we can add it.
			createdAt, _ := time.Parse(time.RFC3339, lpMessage.CreatedAt)
			_, err = b.AddCommentRaw(
				owner,
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
