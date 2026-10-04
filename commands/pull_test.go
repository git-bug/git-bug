package commands

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/repository"
)

func TestPullSuccess(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)
	envA := execenv.NewTestEnvWithRepo(t, repoA)
	envB := execenv.NewTestEnvWithRepo(t, repoB)

	// Create user identity in envA and push to origin
	userA, err := envA.Backend.Identities().New("User A", "a@example.com")
	require.NoError(t, err)
	err = envA.Backend.SetUserIdentity(userA)
	require.NoError(t, err)

	_, err = envA.Backend.Push("origin")
	require.NoError(t, err)

	// Non-verbose pull in envB
	err = runPull(envB, []string{"origin"}, false)
	require.NoError(t, err)

	stdout := envB.Out.String()
	stderr := envB.Err.String()

	require.Contains(t, stdout, "Fetching remote ...")
	require.Contains(t, stdout, "Merging data ...")
	require.Contains(t, stdout, "Summary: 0 new bugs, 0 updated bugs, 1 new identities, 0 updated identities")
	require.NotContains(t, stdout, userA.Id().Human())
	require.Empty(t, stderr)

	// Pull again with non-verbose when there are no new changes
	envB.Out.Reset()
	envB.Err.Reset()

	err = runPull(envB, []string{"origin"}, false)
	require.NoError(t, err)

	stdout = envB.Out.String()
	stderr = envB.Err.String()
	require.Contains(t, stdout, "No new changes")
	require.Empty(t, stderr)

	// Create a new bug in envA and push
	bugA, _, err := envA.Backend.Bugs().New("Pull Bug", "Pull Message")
	require.NoError(t, err)
	_, err = envA.Backend.Push("origin")
	require.NoError(t, err)

	// Verbose pull in envB
	envB.Out.Reset()
	envB.Err.Reset()

	err = runPull(envB, []string{"origin"}, true)
	require.NoError(t, err)

	stdout = envB.Out.String()
	stderr = envB.Err.String()

	require.Empty(t, stdout)
	require.Contains(t, stderr, "Fetching remote ...")
	require.Contains(t, stderr, "Merging data ...")
	require.Contains(t, stderr, bugA.Id().Human())
	require.Contains(t, stderr, "Summary: 1 new bugs, 0 updated bugs, 0 new identities, 0 updated identities")
}

func TestPullFailedMerge(t *testing.T) {
	repoA, repoB, _ := repository.SetupGoGitReposAndRemote(t)
	envA := execenv.NewTestEnvWithRepo(t, repoA)
	envB := execenv.NewTestEnvWithRepo(t, repoB)

	userA, err := envA.Backend.Identities().New("User A", "a@example.com")
	require.NoError(t, err)
	err = envA.Backend.SetUserIdentity(userA)
	require.NoError(t, err)

	bugA, _, err := envA.Backend.Bugs().New("Bug Title", "Bug Message")
	require.NoError(t, err)
	_, err = envA.Backend.Push("origin")
	require.NoError(t, err)

	userB, err := envB.Backend.Identities().New("User B", "b@example.com")
	require.NoError(t, err)
	err = envB.Backend.SetUserIdentity(userB)
	require.NoError(t, err)

	err = runPull(envB, []string{"origin"}, false)
	require.NoError(t, err)

	// Concurrently modify bug on both sides
	bugAResolved, err := envA.Backend.Bugs().Resolve(bugA.Id())
	require.NoError(t, err)
	_, _, err = bugAResolved.AddComment("comment from A")
	require.NoError(t, err)
	require.NoError(t, bugAResolved.Commit())
	_, err = envA.Backend.Push("origin")
	require.NoError(t, err)

	bugBResolved, err := envB.Backend.Bugs().Resolve(bugA.Id())
	require.NoError(t, err)
	_, _, err = bugBResolved.AddComment("comment from B")
	require.NoError(t, err)
	require.NoError(t, bugBResolved.Commit())

	// Clear identity on B so 3-way merge cannot create merge commit
	err = envB.Backend.ClearUserIdentity()
	require.NoError(t, err)

	envB.Out.Reset()
	envB.Err.Reset()

	err = runPull(envB, []string{"origin"}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "1 entity failed to merge")

	stdout := envB.Out.String()
	stderr := envB.Err.String()
	require.Contains(t, stdout, "Summary: 1 failed to merge")
	require.NotContains(t, stdout, "No new changes")
	require.NotEmpty(t, stderr)
}

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
