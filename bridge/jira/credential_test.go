package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestTokenCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotContains(t, r.URL.Path, "//", "transport URLs must not contain a doubled slash")
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
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()
	token := auth.NewToken(target, "api-token")
	token.SetMetadata(auth.MetaKeyLogin, "alice")
	token.SetMetadata(auth.MetaKeyBaseURL, server.URL)
	require.NoError(t, auth.Store(repo, token))
	// Windows clocks can give back-to-back credentials the same timestamp.
	time.Sleep(30 * time.Millisecond)
	password := auth.NewLoginPassword(target, "alice", "account-password")
	password.SetMetadata(auth.MetaKeyLogin, "alice")
	password.SetMetadata(auth.MetaKeyBaseURL, server.URL)
	require.NoError(t, auth.Store(repo, password))
	identity, err := backend.Identities().New("Alice", "alice@example.com")
	require.NoError(t, err)
	identity.SetMetadata(metaKeyJiraLogin, "alice")
	require.NoError(t, identity.Commit())
	conf := core.Configuration{confKeyBaseUrl: server.URL + "/", confKeyDefaultLogin: "alice", confKeyCredentialType: "TOKEN", confKeyProject: "TEST"}
	conf[confKeyCredentialID] = token.ID().String()
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
	credentials, err := listCredentials(repo, "TOKEN", auth.WithTarget(target))
	require.NoError(t, err)
	require.Equal(t, password.ID(), credentials[0].ID(), "legacy unpinned bridges retain newest-secret selection")
	for _, params := range []core.BridgeParams{
		{BaseURL: server.URL + "/", Project: "TEST", CredPrefix: token.ID().String()},
		{BaseURL: server.URL + "/", Project: "TEST", Login: "alice", TokenRaw: "api-token"},
	} {
		configured, err := (&Jira{}).Configure(backend, params, false)
		require.NoError(t, err)
		require.Equal(t, server.URL+"/", configured[confKeyBaseUrl], "preserve the URL used by existing imported issues")
		require.Equal(t, "TOKEN", configured[confKeyCredentialType])
	}
	legacyToken := auth.NewLoginPassword(target, "alice", "api-token")
	legacyToken.SetMetadata(auth.MetaKeyLogin, "alice")
	legacyToken.SetMetadata(auth.MetaKeyBaseURL, server.URL)
	require.NoError(t, auth.Store(repo, legacyToken))
	configured, err := (&Jira{}).Configure(backend, core.BridgeParams{BaseURL: server.URL + "/", Project: "TEST", CredPrefix: legacyToken.ID().String()}, false)
	require.NoError(t, err)
	require.Equal(t, "TOKEN", configured[confKeyCredentialType])
	legacyConf := core.Configuration{confKeyCredentialType: "TOKEN", confKeyCredentialID: password.ID().String()}
	credentials, err = configuredCredentials(repo, legacyConf, auth.WithTarget(target))
	require.NoError(t, err)
	require.Equal(t, password.ID(), credentials[0].ID(), "the validated legacy API credential must remain selected")
	login := auth.NewLogin(target, "alice")
	login.SetMetadata(auth.MetaKeyLogin, "alice")
	login.SetMetadata(auth.MetaKeyBaseURL, server.URL)
	require.NoError(t, auth.Store(repo, login))
	legacyConf[confKeyCredentialID] = login.ID().String()
	credentials, err = configuredCredentials(repo, legacyConf, auth.WithTarget(target))
	require.NoError(t, err)
	require.Equal(t, login.ID(), credentials[0].ID(), "a configured password prompt must not be replaced by a stored secret")
	legacyConf[confKeyCredentialID] = token.ID().String()
	time.Sleep(30 * time.Millisecond)
	replacement := auth.NewToken(target, "replacement")
	replacement.SetMetadata(auth.MetaKeyLogin, "alice")
	replacement.SetMetadata(auth.MetaKeyBaseURL, server.URL)
	require.NoError(t, auth.Store(repo, replacement))
	credentials, err = configuredCredentials(repo, legacyConf, auth.WithTarget(target))
	require.NoError(t, err)
	require.Equal(t, token.ID(), credentials[0].ID(), "adding a token must not silently replace the validated credential")
	legacyConf[confKeyCredentialID] = replacement.ID().String()
	credentials, err = configuredCredentials(repo, legacyConf, auth.WithTarget(target))
	require.NoError(t, err)
	require.Equal(t, replacement.ID(), credentials[0].ID(), "explicit reconfiguration selects the replacement token")
	require.NoError(t, auth.Remove(repo, replacement.ID()))
	_, err = configuredCredentials(repo, legacyConf, auth.WithTarget(target))
	require.ErrorContains(t, err, "reconfigure the bridge")
}

func TestSessionCredentialsIgnoreNewerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/auth/1/session", r.URL.Path)
		var credentials SessionQuery
		require.NoError(t, json.NewDecoder(r.Body).Decode(&credentials))
		require.Equal(t, "alice", credentials.Username)
		require.Equal(t, "password", credentials.Password)
		_, _ = w.Write([]byte(`{"session":{"name":"session","value":"cookie"}}`))
	}))
	defer server.Close()
	repo := repository.NewMockRepo()
	defer repo.Close()
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()
	password := auth.NewLoginPassword(target, "alice", "password")
	token := auth.NewToken(target, "revoked-token")
	for _, credential := range []auth.Credential{password, token} {
		credential.SetMetadata(auth.MetaKeyLogin, "alice")
		credential.SetMetadata(auth.MetaKeyBaseURL, server.URL)
		require.NoError(t, auth.Store(repo, credential))
	}
	identity, err := backend.Identities().New("Alice", "alice@example.com")
	require.NoError(t, err)
	identity.SetMetadata(metaKeyJiraLogin, "alice")
	require.NoError(t, identity.Commit())
	conf := core.Configuration{confKeyBaseUrl: server.URL, confKeyDefaultLogin: "alice", confKeyCredentialType: "SESSION"}
	require.NoError(t, (&jiraImporter{}).Init(context.Background(), backend, conf))
	exporter := &jiraExporter{conf: conf, identityClient: make(map[entity.Id]*Client)}
	require.NoError(t, exporter.cacheAllClient(context.Background(), backend))
	_, err = buildClient(context.Background(), server.URL, "SESSION", token)
	require.ErrorContains(t, err, "requires Jira TOKEN")
}

func TestConfigureRejectsForeignCredentialBeforeNetworkAccess(t *testing.T) {
	for _, targetName := range []string{"github", target} {
		t.Run(targetName, func(t *testing.T) {
			repo := repository.NewMockRepo()
			defer repo.Close()
			backend, err := cache.NewRepoCacheNoEvents(repo, nil)
			require.NoError(t, err)
			defer backend.Close()
			token := auth.NewToken(targetName, "secret")
			token.SetMetadata(auth.MetaKeyLogin, "alice")
			token.SetMetadata(auth.MetaKeyBaseURL, "https://original.example")
			require.NoError(t, auth.Store(repo, token))
			_, err = (&Jira{}).Configure(backend, core.BridgeParams{BaseURL: "https://different.example", Project: "TEST", CredPrefix: token.ID().String()}, false)
			require.Error(t, err)
			if targetName == target {
				require.ErrorContains(t, err, "not bound")
			} else {
				require.ErrorContains(t, err, "not Jira")
			}
		})
	}
}
