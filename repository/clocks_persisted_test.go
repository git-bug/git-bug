package repository

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/util/lamport"
)

func TestPersistedClocks(t *testing.T) {
	// An empty clock file is one being created, or whose creation was
	// interrupted: not a clock yet, and not a reason to fail listing the others.
	t.Run("an empty clock is skipped", func(t *testing.T) {
		storage := NewLocalStorage(t.TempDir())
		clocks := NewPersistedClocks(storage)

		foo, err := clocks.GetOrCreateClock("foo", 1)
		require.NoError(t, err)

		f, err := storage.Create(filepath.Join(clockPath, "empty"))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		allClocks, err := clocks.AllClocks()
		require.NoError(t, err)
		require.Equal(t, map[string]lamport.Clock{"foo": foo}, allClocks)
	})

	// A corrupted clock is listed and handed out, but can't be used until it's
	// created again, at a value the caller vouches for.
	t.Run("a corrupted clock", func(t *testing.T) {
		storage := NewLocalStorage(t.TempDir())
		clocks := NewPersistedClocks(storage)

		clockFile := filepath.Join(clockPath, "foo")
		require.NoError(t, util.WriteFile(storage, clockFile, []byte("garbage"), 0644))

		// listed, for the caller to decide what to do with it
		allClocks, err := clocks.AllClocks()
		require.NoError(t, err)
		require.Contains(t, allClocks, "foo")
		_, err = allClocks["foo"].Time()
		require.ErrorIs(t, err, lamport.ErrClockCorrupt)

		_, err = clocks.Increment("foo")
		require.ErrorIs(t, err, lamport.ErrClockCorrupt)
		require.ErrorIs(t, clocks.Witness("foo", 10), lamport.ErrClockCorrupt)

		clock, err := clocks.GetOrCreateClock("foo", 10)
		require.NoError(t, err)
		time, err := clock.Time()
		require.NoError(t, err)
		require.Equal(t, lamport.Time(10), time)

		// the clock listed earlier is the same one
		time, err = allClocks["foo"].Time()
		require.NoError(t, err)
		require.Equal(t, lamport.Time(10), time)

		time, err = clocks.Increment("foo")
		require.NoError(t, err)
		require.Equal(t, lamport.Time(11), time)
	})
}
