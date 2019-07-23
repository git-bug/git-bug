package gitlab

import (
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// gitlabExporter implement the Exporter interface
type gitlabExporter struct{}

// Init .
func (ge *gitlabExporter) Init(conf core.Configuration) error {
	return nil
}

// ExportAll export all event made by the current user to Gitlab
func (ge *gitlabExporter) ExportAll(repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	return nil, nil
}
