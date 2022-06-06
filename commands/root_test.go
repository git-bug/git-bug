//go:build !windows
// +build !windows

package commands_test

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireGoldenFileEqual(t *testing.T, path string, act []byte) {
	t.Helper()

	// Replace Windows line terminators
	act = bytes.ReplaceAll(act, []byte{'\r'}, []byte{})

	path = filepath.Join("testdata", path)

	if *update {
		require.NoError(t, ioutil.WriteFile(path, act, 0644))
	}

	exp, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(exp), string(act))
}

func TestNewRootCommand(t *testing.T) {
	testEnv := newTestEnv(t)
	testEnv.Execute(t)

	requireGoldenFileEqual(t, "root_out_golden.txt", testEnv.out.Bytes())
}
