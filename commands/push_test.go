package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/execenv/envtest"
	"github.com/git-bug/git-bug/repository/repotest"
)

func TestPushSuccess(t *testing.T) {
	repoA, _, _ := repotest.SetupGoGitReposAndRemote(t)
	envA := envtest.NewEnvWithRepo(t, repoA)

	// Create and set user identity in repoA
	user, err := envA.Backend.Identities().New("Test User", "test@example.com")
	require.NoError(t, err)
	err = envA.Backend.SetUserIdentity(t.Context(), user)
	require.NoError(t, err)

	// Non-verbose push
	err = runPush(t.Context(), envA, []string{"origin"}, false)
	require.NoError(t, err)

	stdout := envA.Out.String()
	stderr := envA.Err.String()
	require.Empty(t, stderr)

	// Reset buffers and create a new bug
	envA.Out.Reset()
	envA.Err.Reset()

	b, _, err := envA.Backend.Bugs().New(t.Context(), "Push Test Bug", "Push Test Message")
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))

	// Verbose push
	err = runPush(t.Context(), envA, []string{"origin"}, true)
	require.NoError(t, err)

	stdout = envA.Out.String()
	stderr = envA.Err.String()
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Pushing to remote: origin")
}

func TestPushVerbose(t *testing.T) {
	env := envtest.NewEnv(t)

	// Test that both verbose and non-verbose handle errors similarly
	// (they return early on error)

	// Test without verbose with no remotes (should fail with some kind of error)
	err := runPush(t.Context(), env, []string{"nonexistent"}, false)
	require.Error(t, err)

	// Non-verbose should fail early without output
	stdout := env.Out.String()
	stderr := env.Err.String()
	require.Empty(t, stdout)
	require.Empty(t, stderr)

	// Reset buffers
	env.Out.Reset()
	env.Err.Reset()

	// Test with verbose with same error
	err = runPush(t.Context(), env, []string{"nonexistent"}, true)
	require.Error(t, err)

	// Verbose mode logs target remote to stderr before attempting push
	stdout = env.Out.String()
	stderr = env.Err.String()
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Pushing to remote: nonexistent")
}

func TestPushDefaultRemote(t *testing.T) {
	env := envtest.NewEnv(t)

	// Test default to origin when no remote specified
	err := runPush(t.Context(), env, []string{}, false)
	require.Error(t, err) // Should fail since no origin remote exists
}

func TestPushTooManyRemotes(t *testing.T) {
	env := envtest.NewEnv(t)

	// Test with too many remotes
	err := runPush(t.Context(), env, []string{"remote1", "remote2"}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Only pushing to one remote at a time is supported")
}
