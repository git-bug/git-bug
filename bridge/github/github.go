package github

import (
	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
)

type Github struct{}

func (*Github) Name() string {
	return "github"
}

func (*Github) Importer() core.Importer {
	return &githubImporter{}
}

func (*Github) Exporter() core.Exporter {
	return nil
}

type githubImporter struct{}

func (*githubImporter) ImportAll(repo *cache.RepoCache, conf core.Configuration) error {
	panic("implement me")
}

func (*githubImporter) Import(repo *cache.RepoCache, conf core.Configuration, id string) error {
	panic("implement me")
}
