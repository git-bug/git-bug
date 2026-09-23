package lamport

import (
	"os"
	"sync"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/require"
)

// The tests below use osfs on purpose: memfs's File.Lock() is an explicit no-op,
// so a test built on it would exercise no locking at all.

func TestPersistedClock(t *testing.T) {
	root := osfs.New(t.TempDir())

	c, err := GetOrCreatePersistedClock(root, "test-clock", 1)
	require.NoError(t, err)

	testClock(t, c)
}

// A clock only comes into existence through GetOrCreatePersistedClock, at the
// value the caller vouches for. Nothing else creates one.
func TestPersistedClockCreation(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		root := osfs.New(t.TempDir())

		_, err := LoadPersistedClock(root, "test-clock")
		require.ErrorIs(t, err, ErrClockNotExist)

		c, err := GetOrCreatePersistedClock(root, "test-clock", 7)
		require.NoError(t, err)
		requireTime(t, c, 7)
	})

	t.Run("empty file", func(t *testing.T) {
		root := osfs.New(t.TempDir())
		require.NoError(t, util.WriteFile(root, "test-clock", nil, 0644))

		_, err := LoadPersistedClock(root, "test-clock")
		require.ErrorIs(t, err, ErrClockNotExist)

		c, err := GetOrCreatePersistedClock(root, "test-clock", 7)
		require.NoError(t, err)
		requireTime(t, c, 7)
	})

	t.Run("zero is not a time", func(t *testing.T) {
		root := osfs.New(t.TempDir())

		_, err := GetOrCreatePersistedClock(root, "test-clock", 0)
		require.Error(t, err)

		_, err = root.Stat("test-clock")
		require.True(t, os.IsNotExist(err))
	})

	t.Run("deleted under a live handle", func(t *testing.T) {
		root := osfs.New(t.TempDir())

		c, err := GetOrCreatePersistedClock(root, "test-clock", 7)
		require.NoError(t, err)
		require.NoError(t, root.Remove("test-clock"))

		_, err = c.Time()
		require.ErrorIs(t, err, ErrClockNotExist)
		_, err = c.Increment()
		require.ErrorIs(t, err, ErrClockNotExist)
		require.ErrorIs(t, c.Witness(42), ErrClockNotExist)

		// and none of that brought the file back
		_, err = root.Stat("test-clock")
		require.True(t, os.IsNotExist(err))

		c, err = GetOrCreatePersistedClock(root, "test-clock", 42)
		require.NoError(t, err)
		requireTime(t, c, 42)
	})
}

// Two clocks over the same file are two processes as far as the file is
// concerned: neither may trust anything it saw earlier.
func TestPersistedClockSharedFile(t *testing.T) {
	tests := []struct {
		name string
		// run a mutation on each clock in turn, and return the value the file
		// is expected to hold afterward
		run func(t *testing.T, c1, c2 *PersistedClock) Time
	}{
		{
			name: "increment sees the other's increments",
			run: func(t *testing.T, c1, c2 *PersistedClock) Time {
				t1, err := c1.Increment()
				require.NoError(t, err)
				require.Equal(t, Time(2), t1)

				// c2 still holds 1 in memory, but the file says 2
				t2, err := c2.Increment()
				require.NoError(t, err)
				require.Equal(t, Time(3), t2)

				t1, err = c1.Increment()
				require.NoError(t, err)
				require.Equal(t, Time(4), t1)

				return 4
			},
		},
		{
			name: "time sees the other's increments",
			run: func(t *testing.T, c1, c2 *PersistedClock) Time {
				for i := 0; i < 5; i++ {
					_, err := c1.Increment()
					require.NoError(t, err)
				}

				// c2 has not been touched since it was loaded holding 1, so
				// this only passes if Time() goes to the file
				requireTime(t, c2, 6)

				return 6
			},
		},
		{
			name: "witness sees the other's increments",
			run: func(t *testing.T, c1, c2 *PersistedClock) Time {
				for i := 0; i < 5; i++ {
					_, err := c1.Increment()
					require.NoError(t, err)
				}
				requireTime(t, c1, 6)

				// lower than the file: must not move it back
				err := c2.Witness(3)
				require.NoError(t, err)
				requireTime(t, c2, 6)

				return 6
			},
		},
		{
			name: "witness is not lost by the other's increment",
			run: func(t *testing.T, c1, c2 *PersistedClock) Time {
				err := c1.Witness(42)
				require.NoError(t, err)

				t2, err := c2.Increment()
				require.NoError(t, err)
				require.Equal(t, Time(43), t2)

				return 43
			},
		},
		{
			name: "creating over an existing clock doesn't move it",
			run: func(t *testing.T, c1, c2 *PersistedClock) Time {
				err := c1.Witness(42)
				require.NoError(t, err)

				// neither down
				c3, err := GetOrCreatePersistedClock(c1.root, c1.filePath, 1)
				require.NoError(t, err)
				requireTime(t, c3, 42)

				// nor up
				c4, err := GetOrCreatePersistedClock(c1.root, c1.filePath, 100)
				require.NoError(t, err)
				requireTime(t, c4, 42)

				return 42
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := osfs.New(t.TempDir())

			c1, err := GetOrCreatePersistedClock(root, "test-clock", 1)
			require.NoError(t, err)
			c2, err := LoadPersistedClock(root, "test-clock")
			require.NoError(t, err)

			expected := tt.run(t, c1, c2)

			require.Equal(t, expected, readClockFile(t, root, "test-clock"))

			// a freshly loaded clock agrees
			c3, err := LoadPersistedClock(root, "test-clock")
			require.NoError(t, err)
			requireTime(t, c3, expected)
		})
	}
}

// Hammer one clock file through several PersistedClock instances at once: no two
// increments may return the same time, and the stored value may never decrease.
func TestPersistedClockConcurrent(t *testing.T) {
	const clocks = 4
	const increments = 50

	root := osfs.New(t.TempDir())

	c, err := GetOrCreatePersistedClock(root, "test-clock", 1)
	require.NoError(t, err)

	all := make([]*PersistedClock, clocks)
	all[0] = c
	for i := 1; i < clocks; i++ {
		all[i], err = LoadPersistedClock(root, "test-clock")
		require.NoError(t, err)
	}

	done := make(chan struct{})
	var watcher sync.WaitGroup
	watcher.Add(1)
	go func() {
		defer watcher.Done()
		var previous Time
		for {
			select {
			case <-done:
				return
			default:
			}
			// t.Error and not require.*, which would stop the wrong goroutine
			c, err := LoadPersistedClock(root, "test-clock")
			if err != nil {
				t.Error(err)
				return
			}
			value, err := c.Time()
			if err != nil {
				t.Error(err)
				return
			}
			if value < previous {
				t.Errorf("the stored clock went backward: %d then %d", previous, value)
				return
			}
			previous = value
		}
	}()

	var wg sync.WaitGroup
	issued := make([][]Time, clocks)
	for i, clock := range all {
		wg.Add(1)
		go func(i int, clock *PersistedClock) {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				time, err := clock.Increment()
				if err != nil {
					// require.* from a goroutine would stop the wrong one
					t.Error(err)
					return
				}
				issued[i] = append(issued[i], time)
			}
		}(i, clock)
	}
	wg.Wait()

	close(done)
	watcher.Wait()

	seen := make(map[Time]bool, clocks*increments)
	var highest Time
	for _, times := range issued {
		for _, time := range times {
			require.False(t, seen[time], "time %d issued twice", time)
			seen[time] = true
			if time > highest {
				highest = time
			}
		}
	}

	require.Len(t, seen, clocks*increments)
	require.Equal(t, highest, readClockFile(t, root, "test-clock"))
}

func readClockFile(t *testing.T, root billy.Filesystem, path string) Time {
	t.Helper()

	c, err := LoadPersistedClock(root, path)
	require.NoError(t, err)

	value, err := c.Time()
	require.NoError(t, err)

	return value
}

func requireTime(t *testing.T, c *PersistedClock, expected Time) {
	t.Helper()

	actual, err := c.Time()
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// requireCorrupt checks that a clock reports ErrClockCorrupt from every
// operation, and that none of them touched the file.
func requireCorrupt(t *testing.T, root billy.Filesystem, c *PersistedClock) {
	t.Helper()

	before, err := util.ReadFile(root, c.filePath)
	require.NoError(t, err)

	_, err = c.Time()
	require.ErrorIs(t, err, ErrClockCorrupt)
	_, err = c.Increment()
	require.ErrorIs(t, err, ErrClockCorrupt)
	require.ErrorIs(t, c.Witness(1_000_000), ErrClockCorrupt)

	after, err := util.ReadFile(root, c.filePath)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

// An interrupted write leaves the start of the new content over the rest of the
// old one. Whatever the cut, including where 99 gains a digit, the clock must
// read as corrupted rather than as some other value.
func TestPersistedClockTornWrite(t *testing.T) {
	newContent := formatClock(100)

	for cut := 1; cut < len(newContent); cut++ {
		root := osfs.New(t.TempDir())

		c, err := GetOrCreatePersistedClock(root, "test-clock", 1)
		require.NoError(t, err)
		require.NoError(t, c.Witness(99))

		f, err := root.OpenFile("test-clock", os.O_WRONLY, 0644)
		require.NoError(t, err)
		_, err = f.Write(newContent[:cut])
		require.NoError(t, err)
		require.NoError(t, f.Close())

		// nothing issues a time from a corrupted clock, and nothing repairs
		// it on the side
		requireCorrupt(t, root, c)

		// a corrupted clock is still handed out, for the caller to decide
		loaded, err := LoadPersistedClock(root, "test-clock")
		require.NoError(t, err)
		requireCorrupt(t, root, loaded)

		// only a creation brings it back, at the value the caller vouches for
		c, err = GetOrCreatePersistedClock(root, "test-clock", 99)
		require.NoError(t, err, "cut at %d", cut)
		requireTime(t, c, 99)
		requireTime(t, loaded, 99)
	}
}

// Clocks written before the checksum hold a bare value. Nothing tells those
// apart from a torn write, so they're corrupted as well, and get rebuilt.
func TestPersistedClockBareValue(t *testing.T) {
	root := osfs.New(t.TempDir())
	require.NoError(t, util.WriteFile(root, "test-clock", []byte("42"), 0644))

	c, err := LoadPersistedClock(root, "test-clock")
	require.NoError(t, err)
	requireCorrupt(t, root, c)

	c, err = GetOrCreatePersistedClock(root, "test-clock", 42)
	require.NoError(t, err)
	requireTime(t, c, 42)

	content, err := util.ReadFile(root, "test-clock")
	require.NoError(t, err)
	require.Equal(t, formatClock(42), content)
}
