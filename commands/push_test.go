package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/execenv"
)

func TestPushVerbose(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test that both verbose and non-verbose handle errors similarly
	// (they return early on error)

	// Test without verbose with no remotes (should fail with some kind of error)
	err := runPush(env, []string{"nonexistent"}, false)
	require.Error(t, err)

	// Reset buffers
	env.Out.Reset()
	env.Err.Reset()

	// Test with verbose with same error
	err = runPush(env, []string{"nonexistent"}, true)
	require.Error(t, err)

	// Both should fail early without much output
	stdout := env.Out.String()
	stderr := env.Err.String()
	require.Empty(t, stdout)
	require.Empty(t, stderr)
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
