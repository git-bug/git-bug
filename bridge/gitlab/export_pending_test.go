package gitlab

import (
	"net/http"
	"net/http/httptest"
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
	require.Equal(t, core.ReasonPendingNoTokenActor, (<-results).Reason)

	require.NoError(t, backend.SetUserIdentity(t.Context(), author))
	_, _, err = b.EditCreateCommentRaw(author, time.Now().Unix(), "New body", nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"iid":1}`))
	}))
	defer server.Close()
	client, err := gitlab.NewClient("token", gitlab.WithBaseURL(server.URL))
	require.NoError(t, err)
	exporter.identityClient[author.Id()] = client
	exporter.repositoryID = "123"
	exporter.exportBug(t.Context(), b, results)
	updated := <-results
	require.NoError(t, updated.Err)
	require.Equal(t, core.ExportEventCommentEdition, updated.Event)
	require.Equal(t, core.ReasonPendingNoTokenActor, (<-results).Reason)
	require.Equal(t, 1, requests)
}
