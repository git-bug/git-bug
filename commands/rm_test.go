package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRm(t *testing.T) {
	testEnv, bugID, _ := newTestEnvAndBug(t)

	require.NoError(t, runRm(testEnv.env, []string{bugID}))
	require.Equal(t, bugID, testEnv.out.String())
	require.Equal(t, "bug  removed\n", testEnv.err.String())
	testEnv.out.Reset()
}
