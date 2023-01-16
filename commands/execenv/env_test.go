package execenv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetIOMode(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)

	testcases := []struct {
		name       string
		in         *os.File
		out        *os.File
		expInMode  IOMode
		expOutMode IOMode
	}{
		{
			name:       "neither redirected",
			in:         os.Stdin,
			out:        os.Stdout,
			expInMode:  TerminalIOMode,
			expOutMode: TerminalIOMode,
		},
		{
			name:       "in redirected",
			in:         w,
			out:        os.Stdout,
			expInMode:  TerminalIOMode,
			expOutMode: TerminalIOMode,
		},
		{
			name:       "out redirected",
			in:         os.Stdin,
			out:        r,
			expInMode:  TerminalIOMode,
			expOutMode: TerminalIOMode,
		},
		{
			name:       "both redirected",
			in:         w,
			out:        r,
			expInMode:  PipedOrRedirectedIOMode,
			expOutMode: PipedOrRedirectedIOMode,
		},
	}

	for i := range testcases {
		testcase := testcases[i]

		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			env := NewEnv()
			require.NotNil(t, env)
		})
	}
}
