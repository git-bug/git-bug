package gitlab

import (
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
)

const (
	keyGitlabLogin = "gitlab-login"
)

type gitlabImporter struct {
	conf core.Configuration

	// number of imported issues
	importedIssues int

	// number of imported identities
	importedIdentities int
}

func (*gitlabImporter) Init(conf core.Configuration) error {
	return nil
}

func (*gitlabImporter) ImportAll(repo *cache.RepoCache, since time.Time) error {
	return nil
}
