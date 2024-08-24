package bugcmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestBugNew(t *testing.T) {
	env, _ := testenv.NewTestEnvAndUser(t)

	err := runBugNew(env, bugNewOptions{
		nonInteractive: true,
		message:        "message",
		title:          "title",
	})
	require.NoError(t, err)
	require.Regexp(t, "^[0-9A-Fa-f]{7} created\n$", env.Out.String())
}
