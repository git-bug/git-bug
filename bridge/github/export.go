package github

import (
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
)

// githubImporter implement the Importer interface
type githubExporter struct {
	conf core.Configuration
}

func (ge *githubExporter) Init(conf core.Configuration) error {
	ge.conf = conf
	return nil
}

// ExportAll export all event made by the current user to Github
func (ge *githubExporter) ExportAll(repo *cache.RepoCache, since time.Time) error {
	identity, err := repo.GetUserIdentity()
	if err != nil {
		return err
	}

	allBugsIds := repo.AllBugsIds()

	//
	bugs := make([]*cache.BugCache, 0)
	for _, id := range allBugsIds {
		b, err := repo.ResolveBug(id)
		if err != nil {
			return err
		}

		// check if user participated in the issue
		participants := b.Snapshot().Participants
		for _, p := range participants {
			if p.Id() == identity.Id() {
				bugs = append(bugs, b)
			}
		}
	}

	//TODO: Export bugs/events/editions

	return nil
}
