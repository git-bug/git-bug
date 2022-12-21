package testenv

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/entity"
)

const (
	testUserName  = "John Doe"
	testUserEmail = "jdoe@example.com"
)

func NewTestEnvAndUser(t *testing.T) (*execenv.Env, entity.Id) {
	t.Helper()

	testEnv := execenv.NewTestEnv(t)

	i, err := testEnv.Backend.Identities().New(testUserName, testUserEmail)
	require.NoError(t, err)

	err = testEnv.Backend.SetUserIdentity(i)
	require.NoError(t, err)

	return testEnv, i.Id()
}

const (
	testBugTitle   = "this is a bug title"
	testBugMessage = "this is a bug message"
)

func NewTestEnvAndBug(t *testing.T) (*execenv.Env, entity.Id) {
	t.Helper()

	testEnv, _ := NewTestEnvAndUser(t)

	b, _, err := testEnv.Backend.Bugs().New(testBugTitle, testBugMessage)
	require.NoError(t, err)

	return testEnv, b.Id()
}

const (
	testCommentMessage = "this is a bug comment"
)

func NewTestEnvAndBugWithComment(t *testing.T) (*execenv.Env, entity.Id, entity.CombinedId) {
	t.Helper()

	env, bugID := NewTestEnvAndBug(t)

	b, err := env.Backend.Bugs().Resolve(bugID)
	require.NoError(t, err)

	commentId, _, err := b.AddComment(testCommentMessage)
	require.NoError(t, err)

	return env, bugID, commentId
}
