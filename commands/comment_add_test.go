package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestEnvAndBugWithComment(t *testing.T) (*testEnv, string, string) {
	t.Helper()

	env, bugID := newTestEnvAndBug(t)

	opts := commentAddOptions{
		message: "this is a bug comment",
	}
	require.NoError(t, runCommentAdd(env.env, opts, []string{bugID}))
	require.NoError(t, runComment(env.env, []string{bugID}))
	comments := parseComments(t, env)
	require.Len(t, comments, 2)

	env.out.Reset()

	return env, bugID, comments[1].id
}

func TestCommentAdd(t *testing.T) {
	const golden = "testdata/comment/add"

	env, bugID, _ := newTestEnvAndBugWithComment(t)
	require.NoError(t, runComment(env.env, []string{bugID}))
	requireCommentsEqual(t, golden, env)
}
