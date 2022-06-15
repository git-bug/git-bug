package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRm(t *testing.T) {
	testEnv, _, bugID := newTestEnvUserAndBug(t)

	exp := "bug " + bugID + " removed\n"

	require.NoError(t, runRm(testEnv.env, []string{bugID}))
	require.Equal(t, exp, testEnv.out.String())
	testEnv.out.Reset()
}
