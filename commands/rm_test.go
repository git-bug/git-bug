package commands_test

import "testing"

func TestRm(t *testing.T) {
	testEnv, _, bugID := newTestEnvUserAndBug(t)

	testEnv.cmd.SetArgs([]string{
		"rm",
		bugID,
	})

	testEnv.Execute(t)
	// TODO: add assertions after #778 is diagnosed and fixed
}
