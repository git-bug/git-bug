package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testUserName  = "John Doe"
	testUserEmail = "jdoe@example.com"
)

func newTestEnvAndUser(t *testing.T) (*testEnv, string) {
	t.Helper()

	testEnv := newTestEnv(t)

	opts := createUserOptions{
		name:           testUserName,
		email:          testUserEmail,
		avatarURL:      "",
		nonInteractive: true,
	}

	require.NoError(t, runUserCreate(testEnv.env, opts))

	userID := strings.TrimSpace(testEnv.out.String())
	testEnv.out.Reset()

	return testEnv, userID
}

func TestUserCreateCommand(t *testing.T) {
	_, userID := newTestEnvAndUser(t)
	require.Regexp(t, "[0-9a-f]{64}", userID)
}
