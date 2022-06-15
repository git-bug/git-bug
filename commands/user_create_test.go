package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestEnvAndUser(t *testing.T) (*testEnv, string) {
	t.Helper()

	testEnv := newTestEnv(t)

	opts := createUserOptions{
		name:           "John Doe",
		email:          "jdoe@example.com",
		avatarURL:      "",
		nonInteractive: true,
	}

	require.NoError(t, runUserCreate(testEnv.env, opts))

	userID := strings.TrimSpace(testEnv.out.String())
	testEnv.out.Reset()

	return testEnv, userID
}

func TestUserCreateCommand(t *testing.T) {
	testEnv, userID := newTestEnvAndUser(t)

	require.FileExists(t, filepath.Join(testEnv.cwd, ".git", "refs", "identities", userID))
	require.FileExists(t, filepath.Join(testEnv.cwd, ".git", "git-bug", "identity-cache"))
}
