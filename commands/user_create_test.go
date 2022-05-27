package commands_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestEnvAndUser(t *testing.T) (*testEnv, string) {
	t.Helper()

	testEnv := newTestEnv(t)

	testEnv.cmd.SetArgs(
		[]string{
			"user",
			"create",
			"--non-interactive",
			"-n John Doe",
			"-e jdoe@example.com",
		})

	testEnv.Execute(t)

	return testEnv, strings.TrimSpace(testEnv.out.String())
}

func TestUserCreateCommand(t *testing.T) {
	testEnv, userID := newTestEnvAndUser(t)

	t.Log("CWD:", testEnv.cwd)

	require.FileExists(t, filepath.Join(testEnv.cwd, ".git", "refs", "identities", userID))
	require.FileExists(t, filepath.Join(testEnv.cwd, ".git", "git-bug", "identity-cache"))
}
