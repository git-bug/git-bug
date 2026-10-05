package gitea

import (
	"context"
	"fmt"
	"time"

	"gitea.dev/sdk"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
)

const (
	target = "gitea"

	metaKeyGiteaID        = "gitea-id"
	metaKeyGiteaCommentID = "gitea-comment-id"
	// metaKeyGiteaEvent identifies the timeline event an operation was
	// imported from, so re-imports don't duplicate it.
	metaKeyGiteaEvent       = "gitea-timeline-event"
	metaKeyGiteaLogin       = "gitea-login"
	metaKeyGiteaScopedLogin = "gitea-instance-login"
	metaKeyGiteaOwner       = "gitea-owner"
	metaKeyGiteaProject     = "gitea-project"
	metaKeyGiteaBaseURL     = "gitea-base-url"

	confKeyOwner        = "owner"
	confKeyProject      = "project"
	confKeyBaseURL      = "base-url"
	confKeyDefaultLogin = "default-login"

	defaultTimeout = 60 * time.Second
)

var _ core.BridgeImpl = &Gitea{}

type Gitea struct{}

func (Gitea) Target() string {
	return target
}

func (g *Gitea) LoginMetaKey() string {
	return metaKeyGiteaLogin
}

func (Gitea) NewImporter() core.Importer {
	return &giteaImporter{}
}

func (Gitea) NewExporter() core.Exporter {
	return &giteaExporter{}
}

func buildClient(baseURL string, token *auth.Token) (*gitea.Client, error) {
	giteaClient, err := gitea.NewClient(baseURL, gitea.SetToken(token.Value))
	if err != nil {
		return nil, err
	}

	return giteaClient, nil
}

// checkIssueTracker fails when the repository's issues do not live in Gitea.
// With issues disabled or redirected to an external tracker, the issue API
// lists nothing and rejects writes, which would otherwise look like an empty
// but working bridge.
func checkIssueTracker(ctx context.Context, client *gitea.Client, owner, project string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	repo, _, err := client.Repositories.GetRepo(ctx, owner, project)
	if err != nil {
		return fmt.Errorf("fetch repository %s/%s: %w", owner, project, err)
	}
	if repo.ExternalTracker != nil {
		return fmt.Errorf("repository %s/%s uses the external issue tracker %s; the Gitea bridge needs the built-in issue tracker",
			owner, project, repo.ExternalTracker.ExternalTrackerURL)
	}
	if !repo.HasIssues {
		return fmt.Errorf("repository %s/%s has issues disabled", owner, project)
	}
	return nil
}
