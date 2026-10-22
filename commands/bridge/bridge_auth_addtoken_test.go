package bridgecmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/todosrht"
	"github.com/git-bug/git-bug/commands/execenv/envtest"
)

func TestAddSourceHutTokenEnablesContributorExport(t *testing.T) {
	env := envtest.NewEnv(t)
	contributor, err := env.Backend.Identities().New("Contributor", "contributor@example.org")
	require.NoError(t, err)
	var submitted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		var request todosrht.GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, submit := request.Variables["input"]; submit {
			submitted = true
			fmt.Fprint(w, `{"data":{"submitTicket":{"id":1}}}`)
		} else {
			fmt.Fprint(w, `{"data":{"me":{"tracker":{"id":1,"name":"issues","owner":{"__typename":"User","canonicalName":"~tester"}}}}}`)
		}
	}))
	defer server.Close()
	options := bridgeAuthAddTokenOptions{target: "todosrht", login: "tester", user: contributor.Id().String(), baseURL: server.URL + "/"}
	require.NoError(t, runBridgeAuthAddToken(env, options, []string{"test-token"}))
	contributor, err = env.Backend.Identities().Resolve(t.Context(), contributor.Id())
	require.NoError(t, err)
	require.Equal(t, "tester", contributor.ImmutableMetadata()["todosrht-login:"+server.URL])
	creds, err := auth.List(env.Backend, auth.WithTarget("todosrht"), auth.WithMeta(auth.MetaKeyBaseURL, server.URL))
	require.NoError(t, err)
	require.Len(t, creds, 1)
	require.NoError(t, env.Backend.SetUserIdentity(t.Context(), contributor))
	b, _, err := env.Backend.Bugs().New(t.Context(), "Contributor's bug", "Body")
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	exporter := (&todosrht.TodoSourceHut{}).NewExporter()
	require.NoError(t, exporter.Init(t.Context(), env.Backend, core.Configuration{
		"base-url": server.URL, "tracker-name": "issues", "default-login": "tester",
	}))
	results, err := exporter.ExportAll(t.Context(), env.Backend, time.Time{})
	require.NoError(t, err)
	for result := range results {
		require.NoError(t, result.Err)
	}
	require.True(t, submitted)

	// One local identity can represent different logins on different instances.
	options.baseURL, options.login = "https://another.sr.ht", "another-login"
	require.NoError(t, runBridgeAuthAddToken(env, options, []string{"another-token"}))
	contributor, err = env.Backend.Identities().Resolve(t.Context(), contributor.Id())
	require.NoError(t, err)
	require.Equal(t, "another-login", contributor.ImmutableMetadata()["todosrht-login:https://another.sr.ht"])
	require.Equal(t, "tester", contributor.ImmutableMetadata()["todosrht-login:"+server.URL])
}

func TestAddSourceHutTokenRejectsDuplicateIdentityMapping(t *testing.T) {
	env := envtest.NewEnv(t)
	imported, err := env.Backend.Identities().NewRaw("Imported", "", "tester", "", nil, map[string]string{
		"todosrht-login:https://todo.sr.ht": "tester",
	})
	require.NoError(t, err)
	local, err := env.Backend.Identities().New("Local", "local@example.org")
	require.NoError(t, err)
	opts := bridgeAuthAddTokenOptions{target: "todosrht", login: "tester", user: local.Id().String(), baseURL: "https://todo.sr.ht"}
	err = runBridgeAuthAddToken(env, opts, []string{"test-token"})
	require.ErrorContains(t, err, "already mapped")
	require.Contains(t, err.Error(), imported.Id().Human())
	local, err = env.Backend.Identities().Resolve(t.Context(), local.Id())
	require.NoError(t, err)
	require.Empty(t, local.ImmutableMetadata())
	creds, err := auth.List(env.Backend, auth.WithTarget("todosrht"))
	require.NoError(t, err)
	require.Empty(t, creds)
	opts.user = imported.Id().String()
	require.NoError(t, runBridgeAuthAddToken(env, opts, []string{"test-token"}))
}

func TestAddTokenValidatesScopeBeforeChangingIdentity(t *testing.T) {
	for _, baseURL := range []string{"", "http://remote.example.org"} {
		t.Run(baseURL, func(t *testing.T) {
			env := envtest.NewEnv(t)
			user, err := env.Backend.Identities().New("Tester", "tester@example.org")
			require.NoError(t, err)
			require.NoError(t, env.Backend.SetUserIdentity(t.Context(), user))
			before := user.ImmutableMetadata()
			require.Error(t, runBridgeAuthAddToken(env, bridgeAuthAddTokenOptions{target: "todosrht", login: "tester", baseURL: baseURL}, []string{"test-token"}))
			require.Equal(t, before, user.ImmutableMetadata())
			creds, err := auth.List(env.Backend)
			require.NoError(t, err)
			require.Empty(t, creds)
		})
	}
}

func TestAddTokenPreservesOtherBridges(t *testing.T) {
	env := envtest.NewEnv(t)
	user, err := env.Backend.Identities().New("Tester", "tester@example.org")
	require.NoError(t, err)
	require.NoError(t, env.Backend.SetUserIdentity(t.Context(), user))
	options := bridgeAuthAddTokenOptions{target: "github", login: "tester"}
	require.NoError(t, runBridgeAuthAddToken(env, options, []string{"test-token"}))
	user, err = env.Backend.Identities().Resolve(t.Context(), user.Id())
	require.NoError(t, err)
	require.Equal(t, "tester", user.ImmutableMetadata()["github-login"])
	// Replacing a credential for the same login must not require an identity edit.
	require.NoError(t, runBridgeAuthAddToken(env, options, []string{"replacement-token"}))
	options.baseURL = "https://unsupported.example.org"
	require.ErrorContains(t, runBridgeAuthAddToken(env, options, []string{"test-token"}), "does not support --base-url")
}
