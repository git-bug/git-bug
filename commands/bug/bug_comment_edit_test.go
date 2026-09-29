package bugcmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestBugCommentEdit(t *testing.T) {
	const golden = "testdata/comment/edit"

	env, bugID, commentID := testenv.NewTestEnvAndBugWithComment(t)

	opts := bugCommentEditOptions{
		message: "this is an altered bug comment",
	}
	require.NoError(t, runBugCommentEdit(env, opts, []string{commentID.Human()}))

	require.NoError(t, runBugComment(env, []string{bugID.Human()}))
	requireCommentsEqual(t, golden, env)
}
