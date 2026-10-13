package gitlab

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

func TestForeignIssueSkipWithoutCredentials(t *testing.T) {
	repo := repository.NewMockRepo()
	defer repo.Close()
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()
	author, err := backend.Identities().New("Alice", "alice@example.com")
	require.NoError(t, err)
	_, _, err = backend.Bugs().NewRaw(author, time.Now().Unix(), "Foreign", "Body", nil, map[string]string{
		metaKeyGitlabId: "1", metaKeyGitlabBaseUrl: "https://gitlab.com/", metaKeyGitlabProject: "other",
	})
	require.NoError(t, err)
	exporter := &gitlabExporter{conf: core.Configuration{confKeyGitlabBaseUrl: "https://gitlab.com/", confKeyProjectID: "current"}}
	results, err := exporter.ExportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	var count int
	for result := range results {
		require.NoError(t, result.Err)
		require.Equal(t, "issue belongs to another GitLab project (ID other)", result.Reason)
		count++
	}
	require.Equal(t, 1, count)
}
