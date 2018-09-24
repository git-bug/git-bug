package github

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
)

func init() {
	core.Register(&Github{})
}

type Github struct{}

func (*Github) Target() string {
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
	fmt.Println(conf)
	fmt.Println("IMPORT ALL")

	return nil
}

func (*githubImporter) Import(repo *cache.RepoCache, conf core.Configuration, id string) error {
	fmt.Println(conf)
	fmt.Println("IMPORT")

	return nil
}
