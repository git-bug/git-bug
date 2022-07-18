package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testUserName  = "John Doe"
	testUserEmail = "jdoe@example.com"
)

func newTestEnvAndUser(t *testing.T) (*testEnv, string, string) {
	t.Helper()

	testEnv := newTestEnv(t)

	opts := createUserOptions{
		name:           testUserName,
		email:          testUserEmail,
		avatarURL:      "",
		nonInteractive: true,
	}

	require.NoError(t, runUserCreate(testEnv.env, opts))

	userID := testEnv.out.String()
	testEnv.out.Reset()
	humanText := testEnv.err.String()
	testEnv.err.Reset()

	return testEnv, userID, humanText
}

func TestUserCreateCommand(t *testing.T) {
	_, userID, humanText := newTestEnvAndUser(t)
	require.Regexp(t, "^[0-9a-f]{64}$", userID)
	require.Equal(t, "\n\n", humanText)
}
