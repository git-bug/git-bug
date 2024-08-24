package usercmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestUserNewCommand(t *testing.T) {
	_, userID := testenv.NewTestEnvAndUser(t)
	require.Regexp(t, "[0-9a-f]{64}", userID)
}
