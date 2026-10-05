package jira

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestTokenCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		login, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "alice", login)
		require.Equal(t, "api-token", password)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","key":"TEST"}`))
	}))
	defer server.Close()
	repo := repository.NewMockRepo()
	defer repo.Close()
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	defer backend.Close()
	token := auth.NewToken(target, "api-token")
	token.SetMetadata(auth.MetaKeyLogin, "alice")
	token.SetMetadata(auth.MetaKeyBaseURL, server.URL)
	require.NoError(t, auth.Store(repo, token))
	identity, err := backend.Identities().New("Alice", "alice@example.com")
	require.NoError(t, err)
	identity.SetMetadata(metaKeyJiraLogin, "alice")
	require.NoError(t, identity.Commit())
	conf := core.Configuration{confKeyBaseUrl: server.URL + "/", confKeyDefaultLogin: "alice", confKeyCredentialType: "TOKEN", confKeyProject: "TEST"}
	importer := &jiraImporter{}
	require.NoError(t, importer.Init(context.Background(), backend, conf))
	_, err = importer.client.GetProject("TEST")
	require.NoError(t, err)
	exporter := &jiraExporter{conf: conf, identityClient: make(map[entity.Id]*Client)}
	require.NoError(t, exporter.cacheAllClient(context.Background(), backend))
	client, err := exporter.getClientForIdentity(identity.Id())
	require.NoError(t, err)
	_, err = client.GetProject("TEST")
	require.NoError(t, err)
	for _, params := range []core.BridgeParams{
		{BaseURL: server.URL + "/", Project: "TEST", CredPrefix: token.ID().String()},
		{BaseURL: server.URL + "/", Project: "TEST", Login: "alice", TokenRaw: "api-token"},
	} {
		configured, err := (&Jira{}).Configure(backend, params, false)
		require.NoError(t, err)
		require.Equal(t, server.URL+"/", configured[confKeyBaseUrl], "preserve the URL used by existing imported issues")
		require.Equal(t, "TOKEN", configured[confKeyCredentialType])
	}
}
