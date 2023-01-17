package execenv

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTerminal(t *testing.T) {
	// easy way to get a reader and a writer
	r, w, err := os.Pipe()
	require.NoError(t, err)

	require.False(t, isTerminal(r))
	require.False(t, isTerminal(w))

	// golang's testing framework replaces os.Stdin and os.Stdout, so the following doesn't work here
	// require.True(t, isTerminal(os.Stdin))
	// require.True(t, isTerminal(os.Stdout))
}
