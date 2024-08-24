package bugcmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestBugCommentNew(t *testing.T) {
	const golden = "testdata/comment/add"

	env, bugID, _ := testenv.NewTestEnvAndBugWithComment(t)

	require.NoError(t, runBugComment(env, []string{bugID.String()}))
	requireCommentsEqual(t, golden, env)
}
