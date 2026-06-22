package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/execenv"
)

func TestPullVerbose(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test without verbose with no remotes (should fail with some kind of error)
	err := runPull(env, []string{"nonexistent"}, false)
	require.Error(t, err)
	stdout := env.Out.String()
	stderr := env.Err.String()

	// In non-verbose mode, should see normal output on stdout
	require.NotEmpty(t, stdout)
	require.Contains(t, stdout, "Fetching remote ...")
	// Error happens before "Merging data ..." is printed

	// Reset buffers
	env.Out.Reset()
	env.Err.Reset()

	// Test with verbose with same error
	err = runPull(env, []string{"nonexistent"}, true)
	require.Error(t, err)

	stdout = env.Out.String()
	stderr = env.Err.String()

	// In verbose mode, should have verbose output to stderr
	require.Empty(t, stdout)
	require.NotEmpty(t, stderr)
	require.Contains(t, stderr, "Fetching remote ...")
	// Error happens before "Merging data ..." is printed
}

func TestPullOutputModes(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test default to origin when no remote specified (non-verbose)
	err := runPull(env, []string{}, false)
	require.Error(t, err) // Should fail since no origin remote exists

	stdout := env.Out.String()

	// In non-verbose mode, should show normal output on stdout
	require.NotEmpty(t, stdout)
	require.Contains(t, stdout, "Fetching remote ...")
	// Error happens before "Merging data ..." is printed

	// Reset buffers
	env.Out.Reset()
	env.Err.Reset()

	// Test with verbose flag
	err = runPull(env, []string{}, true)
	require.Error(t, err)

	stdout = env.Out.String()
	stderr := env.Err.String()

	// In verbose mode, normal output goes to stderr
	require.Empty(t, stdout)
	require.NotEmpty(t, stderr)
	require.Contains(t, stderr, "Fetching remote ...")
}

func TestPullTooManyRemotes(t *testing.T) {
	env := execenv.NewTestEnv(t)

	// Test with too many remotes
	err := runPull(env, []string{"remote1", "remote2"}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Only pulling from one remote at a time is supported")
}
