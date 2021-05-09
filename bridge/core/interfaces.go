package core

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/cache"
)

type Configuration map[string]string

type BridgeImpl interface {
	// Target return the target of the bridge (e.g.: "github")
	Target() string

	// NewImporter return an Importer implementation if the import is supported
	NewImporter() Importer

	// NewExporter return an Exporter implementation if the export is supported
	NewExporter() Exporter

	// Configure handle the user interaction and return a key/value configuration
	// for future use.
	Configure(repo *cache.RepoCache, params BridgeParams, interactive bool) (Configuration, error)

	// The set of the BridgeParams fields supported
	ValidParams() map[string]interface{}

	// ValidateConfig check the configuration for error
	ValidateConfig(conf Configuration) error

	// LoginMetaKey return the metadata key used to store the remote bug-tracker login
	// on the user identity. The corresponding value is used to match identities and
	// credentials.
	LoginMetaKey() string
}

type Importer interface {
	Init(ctx context.Context, repo *cache.RepoCache, conf Configuration) error
	ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan ImportResult, error)
}

type Exporter interface {
	Init(ctx context.Context, repo *cache.RepoCache, conf Configuration) error
	ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan ExportResult, error)
}
