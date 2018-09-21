package core

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

type Configuration map[string]string

type Importer interface {
	ImportAll(repo *cache.RepoCache, conf Configuration) error
	Import(repo *cache.RepoCache, conf Configuration, id string) error
}

type Exporter interface {
	ExportAll(repo *cache.RepoCache, conf Configuration) error
	Export(repo *cache.RepoCache, conf Configuration, id string) error
}

type BridgeImpl interface {
	Name() string

	// Configure handle the user interaction and return a key/value configuration
	// for future use
	Configure(repo repository.RepoCommon) (Configuration, error)

	Importer() Importer
	Exporter() Exporter
}
