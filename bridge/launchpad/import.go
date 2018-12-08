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
		lpBugID := fmt.Sprintf("%d", lpBug.ID)
		_, err := repo.ResolveBugCreateMetadata(keyLaunchpadID, lpBugID)
		if err != nil && err != bug.ErrBugNotExist {
			return err
		}

		if err == bug.ErrBugNotExist {
			createdAt, _ := time.Parse(time.RFC3339, lpBug.CreatedAt)
			_, err := repo.NewBugRaw(
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

	}
	return nil
}

func (li *launchpadImporter) Import(repo *cache.RepoCache, id string) error {
	fmt.Println("IMPORT")
	return nil
}
