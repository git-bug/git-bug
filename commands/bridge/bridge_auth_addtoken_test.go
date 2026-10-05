package bridgecmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestBridgeAuthAddToken_BaseURLNormalization(t *testing.T) {
	env, _ := testenv.NewTestEnvAndUser(t)

	opts := bridgeAuthAddTokenOptions{
		target:  "gitlab",
		login:   "alice",
		baseURL: "https://gitlab.com/",
	}

	err := runBridgeAuthAddToken(env, opts, []string{"glpat-test-token-12345"})
	require.NoError(t, err)

	// Verify the stored credential has normalized base-url metadata (no trailing slash)
	creds, err := auth.List(env.Repo,
		auth.WithTarget("gitlab"),
		auth.WithMeta(auth.MetaKeyBaseURL, "https://gitlab.com"),
	)
	require.NoError(t, err)
	require.Len(t, creds, 1)

	storedBaseURL, ok := creds[0].GetMetadata(auth.MetaKeyBaseURL)
	require.True(t, ok)
	assert.Equal(t, "https://gitlab.com", storedBaseURL)

	// Verify that searching with trailing slash also matches
	credsWithSlash, err := auth.List(env.Repo,
		auth.WithTarget("gitlab"),
		auth.WithMeta(auth.MetaKeyBaseURL, "https://gitlab.com/"),
	)
	require.NoError(t, err)
	require.Len(t, credsWithSlash, 1)
	assert.Equal(t, creds[0].ID(), credsWithSlash[0].ID())
}

func TestBridgeAuthAddTokenRejectsInvalidBaseURL(t *testing.T) {
	for _, baseURL := range []string{"gitlab.com", "ftp://gitlab.com", "https://user:password@gitlab.com", "https://gitlab.com?token=secret", "https://gitlab.com/#fragment"} {
		t.Run(baseURL, func(t *testing.T) {
			env, _ := testenv.NewTestEnvAndUser(t)
			err := runBridgeAuthAddToken(env, bridgeAuthAddTokenOptions{target: "gitlab", login: "alice", baseURL: baseURL}, []string{"token"})
			require.ErrorContains(t, err, "invalid --base-url")
			creds, err := auth.List(env.Repo)
			require.NoError(t, err)
			require.Empty(t, creds)
		})
	}
}
