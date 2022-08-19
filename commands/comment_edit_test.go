package commands

import (
	"testing"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/stretchr/testify/require"
)

func TestCommentEdit(t *testing.T) {
	const golden = "testdata/comment/edit"

	env, bugID, commentID := newTestEnvAndBugWithComment(t)

	opts := commentEditOptions{
		message: "this is an altered bug comment",
	}
	require.NoError(t, runCommentEdit(env.env, opts, []string{commentID}))

	// TODO: remove this comment and everything between the "snip"
	//       comments when issue #850 is resolved.

	// ***** snip *****
	cache, err := env.env.backend.ResolveBugPrefix(bugID)
	require.NoError(t, err)
	bu, err := bug.Read(env.env.repo, cache.Id())
	require.NoError(t, err)
	for _, op := range bu.Operations() {
		t.Log("Operation: ", op)
	}
	t.Log("Compiled comments: ", bu.Compile().Comments)
	// ***** snip *****

	require.NoError(t, runComment(env.env, []string{bugID}))
	requireCommentsEqual(t, golden, env)
}
