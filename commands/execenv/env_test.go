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
		expInMode  bool
		expOutMode bool
	}{
		{
			name:       "neither redirected",
			in:         os.Stdin,
			out:        os.Stdout,
			expInMode:  false,
			expOutMode: false,
		},
		{
			name:       "in redirected",
			in:         w,
			out:        os.Stdout,
			expInMode:  true,
			expOutMode: false,
		},
		{
			name:       "out redirected",
			in:         os.Stdin,
			out:        r,
			expInMode:  false,
			expOutMode: true,
		},
		{
			name:       "both redirected",
			in:         w,
			out:        r,
			expInMode:  true,
			expOutMode: true,
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
