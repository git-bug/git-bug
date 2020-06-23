package lamport

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistedClock(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	c, err := NewPersistedClock(path.Join(dir, "test-clock"))
	require.NoError(t, err)

	testClock(t, c)
}
