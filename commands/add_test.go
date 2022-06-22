package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestEnvAndBug(t *testing.T) (*testEnv, string) {
	t.Helper()

	testEnv, _ := newTestEnvAndUser(t)
	opts := addOptions{
		title:          "this is a bug title",
		message:        "this is a bug message",
		messageFile:    "",
		nonInteractive: true,
	}

	require.NoError(t, runAdd(testEnv.env, opts))
	require.Regexp(t, "^[0-9A-Fa-f]{7} created\n$", testEnv.out)
	bugID := strings.Split(testEnv.out.String(), " ")[0]
	testEnv.out.Reset()

	return testEnv, bugID
}

func TestAdd(t *testing.T) {
	_, bugID := newTestEnvAndBug(t)
	require.Regexp(t, "^[0-9A-Fa-f]{7}$", bugID)
}
