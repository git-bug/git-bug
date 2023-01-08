package testenv

import (
	"testing"

	"github.com/fatih/color"
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

	// The Go testing framework either uses os.Stdout directly or a buffer
	// depending on how the command is initially launched.  This results
	// in os.Stdout.Fd() sometimes being a Terminal, and other times not
	// being a Terminal which determines whether the ANSI library sends
	// escape sequences to colorize the text.
	//
	// The line below disables all colorization during testing so that the
	// git-bug command output is consistent in all test scenarios.
	//
	// See:
	// - https://github.com/MichaelMure/git-bug/issues/926
	// - https://github.com/golang/go/issues/57671
	// - https://github.com/golang/go/blob/f721fa3be9bb52524f97b409606f9423437535e8/src/cmd/go/internal/test/test.go#L1180-L1208
	// - https://github.com/golang/go/issues/34877
	color.NoColor = true

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
