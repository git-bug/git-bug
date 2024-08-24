package bugcmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestBugRm(t *testing.T) {
	env, bugID := testenv.NewTestEnvAndBug(t)

	exp := "bug " + bugID.Human() + " removed\n"

	require.NoError(t, runBugRm(env, []string{bugID.Human()}))
	require.Equal(t, exp, env.Out.String())
	env.Out.Reset()
}
