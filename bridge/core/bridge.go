package core

import "github.com/MichaelMure/git-bug/cache"

type Common interface {
	// Configure handle the user interaction and return a key/value configuration
	// for future use
	Configure() (map[string]string, error)
}

type Importer interface {
	Common
	ImportAll(repo *cache.RepoCache) error
	Import(repo *cache.RepoCache, id string) error
}

type Exporter interface {
	Common
	ExportAll(repo *cache.RepoCache) error
	Export(repo *cache.RepoCache, id string) error
}

type NotSupportedImporter struct{}
type NotSupportedExporter struct{}

// persist
