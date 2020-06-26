package lamport

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var ErrClockNotExist = errors.New("clock doesn't exist")

type PersistedClock struct {
	*MemClock
	filePath string
}

// NewPersistedClock create a new persisted Lamport clock
func NewPersistedClock(filePath string) (*PersistedClock, error) {
	clock := &PersistedClock{
		MemClock: NewMemClock(),
		filePath: filePath,
	}

	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return nil, err
	}

	err = clock.Write()
	if err != nil {
		return nil, err
	}

	return clock, nil
}

// LoadPersistedClock load a persisted Lamport clock from a file
func LoadPersistedClock(filePath string) (*PersistedClock, error) {
	clock := &PersistedClock{
		filePath: filePath,
	}

	err := clock.read()
	if err != nil {
		return nil, err
	}

	return clock, nil
}

// Increment is used to return the value of the lamport clock and increment it afterwards
func (pc *PersistedClock) Increment() (Time, error) {
	time, err := pc.MemClock.Increment()
	if err != nil {
		return 0, err
	}
	return time, pc.Write()
}

// Witness is called to update our local clock if necessary after
// witnessing a clock value received from another process
func (pc *PersistedClock) Witness(time Time) error {
	// TODO: rework so that we write only when the clock was actually updated
	err := pc.MemClock.Witness(time)
	if err != nil {
		return err
	}
	return pc.Write()
}

func (pc *PersistedClock) read() error {
	content, err := ioutil.ReadFile(pc.filePath)
	if os.IsNotExist(err) {
		return ErrClockNotExist
	}
	if err != nil {
		return err
	}

	var value uint64
	n, err := fmt.Sscanf(string(content), "%d", &value)
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("could not read the clock")
	}

	pc.MemClock = NewMemClockWithTime(value)

	return nil
}

func (pc *PersistedClock) Write() error {
	data := []byte(fmt.Sprintf("%d", pc.counter))
	return ioutil.WriteFile(pc.filePath, data, 0644)
}
