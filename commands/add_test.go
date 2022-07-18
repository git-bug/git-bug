package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestEnvAndBug(t *testing.T) (*testEnv, string, string) {
	t.Helper()

	testEnv, _, _ := newTestEnvAndUser(t)
	opts := addOptions{
		title:          "this is a bug title",
		message:        "this is a bug message",
		messageFile:    "",
		nonInteractive: true,
	}

	require.NoError(t, runAdd(testEnv.env, opts))
	// require.Regexp(t, "^[0-9A-Fa-f]{7} created\n$", testEnv.out)
	// bugID := strings.Split(testEnv.out.String(), " ")[0]
	bugID := testEnv.out.String()
	testEnv.out.Reset()
	humanText := testEnv.err.String()
	testEnv.err.Reset()

	return testEnv, bugID, humanText
}

func TestAdd(t *testing.T) {
	_, bugID, humanText := newTestEnvAndBug(t)
	require.Regexp(t, "^[0-9A-Fa-f]{7}$", bugID)
	require.Equal(t, " created\n", humanText)
}
