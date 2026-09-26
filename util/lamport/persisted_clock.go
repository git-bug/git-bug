package lamport

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-git/go-billy/v5"
)

// ErrClockNotExist is returned when there is no clock: no file, or an empty one
// left by a creation that didn't complete.
var ErrClockNotExist = errors.New("clock doesn't exist")

// ErrClockCorrupt is returned for a clock file holding something else than a
// clock, typically what an interrupted write left behind.
var ErrClockCorrupt = errors.New("clock is corrupted")

var _ Clock = &PersistedClock{}

// PersistedClock is a Lamport clock stored in a file, safe to use concurrently
// from several processes and without any repository-wide coordination.
//
// The file is the only state: a read goes to it, and a mutation is a
// read-modify-write cycle, both under a lock on it. So a time from Increment is
// greater than every time this clone issued or witnessed before, the stored
// value never decreases, and a Witness is visible to every later reader.
//
// A value is not tied to what the caller does with it though: it's stale as soon
// as the lock is released, and the commit carrying it lands later. Increment
// reserves a position in the order, Time only observes one.
//
// A clock that is missing or corrupted has lost that guarantee, so nothing here
// quietly brings one back: Time, Increment and Witness all fail on it, with
// ErrClockNotExist or ErrClockCorrupt, and only GetOrCreatePersistedClock
// creates one, at a value the caller vouches for.
//
// Nothing is flushed to disk. Readers go through the page cache so they always
// see the current value, but a power loss can take the file back - deliberately,
// as git doesn't sync the commit carrying the time either, and Witness pulls a
// clock that came back low up again when entities are read.
type PersistedClock struct {
	root     billy.Filesystem
	filePath string

	// the file lock would serialize our own goroutines too, being held per open
	// file description rather than per process, but this is cheaper than
	// contending in the kernel and still holds where Lock() lies, as billy's
	// osfs does on Plan 9
	mu sync.Mutex
}

// GetOrCreatePersistedClock return the Lamport clock stored in that file,
// creating it at the initial value if there is none, or if what's there is
// corrupted. An existing clock is left alone, whatever the initial value:
// another process can create it at any moment, and overwriting what we find
// would take it backward.
//
// The initial value must hold the guarantee documented on PersistedClock: it has
// to be at least every time this clone already issued or witnessed, which for a
// clock ordering entities means reading them all first.
func GetOrCreatePersistedClock(root billy.Filesystem, filePath string, initial Time) (*PersistedClock, error) {
	if initial == 0 {
		return nil, fmt.Errorf("lamport: 0 is not a valid time")
	}

	err := root.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return nil, err
	}

	clock := &PersistedClock{root: root, filePath: filePath}

	err = clock.create(initial)
	if err != nil {
		return nil, err
	}

	return clock, nil
}

// LoadPersistedClock load a persisted Lamport clock from a file, and return
// ErrClockNotExist if there is none. A corrupted clock is still one: it's
// returned, and reports ErrClockCorrupt when used, for the caller to decide.
func LoadPersistedClock(root billy.Filesystem, filePath string) (*PersistedClock, error) {
	clock := &PersistedClock{root: root, filePath: filePath}

	// not to hold the value - every use reads the file - but to report a
	// missing clock
	_, err := clock.read()
	if err != nil && !errors.Is(err, ErrClockCorrupt) {
		return nil, err
	}

	return clock, nil
}

// Time is used to return the current value of the lamport clock
func (pc *PersistedClock) Time() (Time, error) {
	return pc.read()
}

// Increment is used to return the value of the lamport clock and increment it afterwards
func (pc *PersistedClock) Increment() (Time, error) {
	var incremented Time

	err := pc.update(func(current Time) (Time, error) {
		if current == math.MaxUint64 {
			return 0, ErrClockOverflow
		}
		incremented = current + 1
		return incremented, nil
	})
	if err != nil {
		return 0, err
	}

	return incremented, nil
}

// Witness is called to update our local clock if necessary after
// witnessing a clock value received from another process
func (pc *PersistedClock) Witness(time Time) error {
	return pc.update(func(current Time) (Time, error) {
		if time <= current {
			// already past that value, update() writes nothing
			return current, nil
		}
		return time, nil
	})
}

// create writes the initial value if the file holds no usable clock, and leaves
// it alone otherwise.
func (pc *PersistedClock) create(initial Time) error {
	return pc.locked(os.O_RDWR|os.O_CREATE, func(f billy.File) error {
		_, err := readClock(f)
		if errors.Is(err, ErrClockNotExist) || errors.Is(err, ErrClockCorrupt) {
			return writeClock(f, initial)
		}
		return err
	})
}

// update runs a mutation with the file locked, so that the whole
// read-modify-write cycle is atomic against the other processes doing the same.
// mutate gets the value read from the file and returns the new one, which must
// not be lower - update panics otherwise; returning it unchanged writes nothing. A clock that is missing
// or corrupted fails, and is left as it is, as does a failing mutate.
func (pc *PersistedClock) update(mutate func(current Time) (Time, error)) error {
	return pc.locked(os.O_RDWR, func(f billy.File) error {
		current, err := readClock(f)
		if err != nil {
			return err
		}

		value, err := mutate(current)
		if err != nil {
			return err
		}
		if value == current {
			return nil
		}
		if value < current {
			panic(fmt.Sprintf("lamport: clock going from %d back to %d", current, value))
		}

		return writeClock(f, value)
	})
}

// read the stored value, locking the file so we don't catch a write in progress
func (pc *PersistedClock) read() (Time, error) {
	var value Time

	err := pc.locked(os.O_RDONLY, func(f billy.File) (err error) {
		value, err = readClock(f)
		return err
	})

	return value, err
}

// locked opens the file, locks it, runs fn on it, then unlocks and closes it.
// A missing file gives ErrClockNotExist, unless flag asks to create it.
func (pc *PersistedClock) locked(flag int, fn func(f billy.File) error) (err error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	f, err := pc.root.OpenFile(pc.filePath, flag, 0644)
	if os.IsNotExist(err) {
		return ErrClockNotExist
	}
	if err != nil {
		return err
	}

	err = f.Lock()
	if err != nil {
		_ = f.Close()
		return err
	}

	defer func() {
		unlockErr := f.Unlock()
		closeErr := f.Close()
		if err == nil {
			err = errors.Join(unlockErr, closeErr)
		}
	}()

	return fn(f)
}

// readClock read a value from an open, locked file. An empty file gives
// ErrClockNotExist: that's not a clock, only the trace of a creation.
func readClock(f billy.File) (Time, error) {
	_, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}

	if len(content) == 0 {
		return 0, ErrClockNotExist
	}

	value, ok := parseClock(content)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrClockCorrupt, f.Name())
	}

	return value, nil
}

// writeClock store a value in an open, locked file. In place rather than through
// a rename: a rename would leave us locking an inode nobody reads anymore while
// the next writer locks the new one, so no exclusion at all. Truncating after
// the write keeps the file from ever being seen empty, which would read as no
// clock at all.
func writeClock(f billy.File, value Time) error {
	data := formatClock(value)

	_, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return f.Truncate(int64(len(data)))
}

// formatClock gives the value followed by its crc32, as "<value> <crc32>\n".
// Writing in place, an interrupted write leaves the start of the new content
// over the rest of the old one: "99" becoming "100" can end up as "19", a valid
// and lower value. The checksum turns that into a detectable corruption.
func formatClock(value Time) []byte {
	digits := strconv.FormatUint(uint64(value), 10)
	return []byte(fmt.Sprintf("%s %08x\n", digits, crc32.ChecksumIEEE([]byte(digits))))
}

// parseClock read what formatClock wrote, and nothing else: content that doesn't
// format back to exactly itself is corrupted. That includes the bare value
// written before the checksum, which gets rebuilt like any corrupted clock.
func parseClock(content []byte) (Time, bool) {
	digits, _, _ := strings.Cut(string(content), " ")

	value, err := strconv.ParseUint(digits, 10, 64)
	if err != nil || value == 0 {
		// 0 is not a valid time
		return 0, false
	}

	return Time(value), bytes.Equal(content, formatClock(Time(value)))
}
