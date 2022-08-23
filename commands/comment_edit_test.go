package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommentEdit(t *testing.T) {
	const golden = "testdata/comment/edit"

	env, bugID, commentID := newTestEnvAndBugWithComment(t)

	opts := commentEditOptions{
		message: "this is an altered bug comment",
	}
	require.NoError(t, runCommentEdit(env.env, opts, []string{commentID}))

	require.NoError(t, runComment(env.env, []string{bugID}))
	requireCommentsEqual(t, golden, env)
}
