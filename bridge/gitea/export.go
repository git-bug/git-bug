package gitea

import (
	"context"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
)

var (
	ErrMissingIdentityToken = errors.New("missing identity token")
)

// giteaExporter implement the Exporter interface
type giteaExporter struct {
}

// Init .
func (ge *giteaExporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	return syscall.ENOSYS
}

func (ge *giteaExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	return nil, syscall.ENOSYS
}
