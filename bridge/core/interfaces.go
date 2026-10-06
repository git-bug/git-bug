package core

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/cache"
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

// ExportOptions adjusts a single push.
type ExportOptions struct {
	// Foreign also pushes bugs that originate from another bug tracker.
	Foreign bool
}

// ForeignExporter is implemented by exporters that can mirror bugs coming
// from another bug tracker. EnableForeign is called before Init.
//
// A foreign push must not mix mirrored bugs with unrelated remote issues.
// Init must therefore fail unless every issue in the remote tracker was
// created by mirroring (an empty tracker trivially qualifies).
//
// Mirrored bugs record the remote identifiers under the keys returned by
// MirrorMetaKey, so they never clash with the metadata of the bridge the bug
// was imported from. Operations by other authors are pushed as the token
// owner, prefixed with MirrorAttribution.
type ForeignExporter interface {
	EnableForeign()
}
