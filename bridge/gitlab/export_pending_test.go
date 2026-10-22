package gitlab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestPendingChangeWithoutTokenIsReported(t *testing.T) {
	backend, err := cache.NewRepoCacheNoEvents(repotest.NewGoGitRepo(t, false), nil)
	require.NoError(t, err)
	defer backend.Close()
	author, err := backend.Identities().New("Author", "author@example.org")
	require.NoError(t, err)
	other, err := backend.Identities().New("Other", "other@example.org")
	require.NoError(t, err)
	b, _, err := backend.Bugs().NewRaw(author, time.Now().Unix(), "Mapped", "Body", nil, map[string]string{
		metaKeyGitlabId: "1", metaKeyGitlabBaseUrl: "https://gitlab.com/", metaKeyGitlabProject: "123",
	})
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	exporter := &gitlabExporter{conf: core.Configuration{confKeyGitlabBaseUrl: "https://gitlab.com/", confKeyProjectID: "123"},
		identityClient: map[entity.Id]*gitlab.Client{author.Id(): nil}, cachedOperationIDs: map[string]string{}}
	results := make(chan core.ExportResult, 10)
	exporter.exportBug(t.Context(), b, results)
	require.Equal(t, core.ReasonNothingExported, (<-results).Reason)
	_, _, err = b.AddCommentRaw(other, time.Now().Unix(), "Pending", nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	exporter.exportBug(t.Context(), b, results)
	require.Equal(t, core.ReasonNoTokenActor, (<-results).Reason)
}
