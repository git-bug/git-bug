package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/repository"
)

func TestPushSuccess(t *testing.T) {
	repoA, _, _ := repository.SetupGoGitReposAndRemote(t)
	envA := execenv.NewTestEnvWithRepo(t, repoA)

	// Create and set user identity in repoA
	user, err := envA.Backend.Identities().New("Test User", "test@example.com")
	require.NoError(t, err)
	err = envA.Backend.SetUserIdentity(user)
	require.NoError(t, err)

	// Non-verbose push
	err = runPush(envA, []string{"origin"}, false)
	require.NoError(t, err)

	stdout := envA.Out.String()
	stderr := envA.Err.String()
	require.Empty(t, stderr)

	// Reset buffers and create a new bug
	envA.Out.Reset()
	envA.Err.Reset()

	_, _, err = envA.Backend.Bugs().New("Push Test Bug", "Push Test Message")
	require.NoError(t, err)

	// Verbose push
	err = runPush(envA, []string{"origin"}, true)
	require.NoError(t, err)

	stdout = envA.Out.String()
	stderr = envA.Err.String()
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Pushing to remote: origin")
}

func TestPushVerbose(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test that both verbose and non-verbose handle errors similarly
	// (they return early on error)

	// Test without verbose with no remotes (should fail with some kind of error)
	err := runPush(env, []string{"nonexistent"}, false)
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
	err = runPush(env, []string{"nonexistent"}, true)
	require.Error(t, err)

	// Verbose mode logs target remote to stderr before attempting push
	stdout = env.Out.String()
	stderr = env.Err.String()
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Pushing to remote: nonexistent")
}

func TestPushDefaultRemote(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test default to origin when no remote specified
	err := runPush(env, []string{}, false)
	require.Error(t, err) // Should fail since no origin remote exists
}

func TestPushTooManyRemotes(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test with too many remotes
	err := runPush(env, []string{"remote1", "remote2"}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Only pushing to one remote at a time is supported")
}
