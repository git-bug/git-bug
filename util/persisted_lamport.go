package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type PersistedLamport struct {
	LamportClock
	filePath string
}

func NewPersistedLamport(filePath string) *PersistedLamport {
	clock := &PersistedLamport{
		LamportClock: NewLamportClock(),
		filePath:     filePath,
	}
	return clock
}

func LoadPersistedLamport(filePath string) (*PersistedLamport, error) {
	clock := &PersistedLamport{
		filePath: filePath,
	}

	err := clock.read()
	if err != nil {
		return nil, err
	}

	return clock, nil
}

func (c *PersistedLamport) Increment() (LamportTime, error) {
	time := c.LamportClock.Increment()
	return time, c.Write()
}

func (c *PersistedLamport) Witness(time LamportTime) error {
	// TODO: rework so that we write only when the clock was actually updated
	c.LamportClock.Witness(time)
	return c.Write()
}

func (c *PersistedLamport) read() error {
	content, err := ioutil.ReadFile(c.filePath)
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

	c.LamportClock = NewLamportClockWithTime(value)

	return nil
}

func (c *PersistedLamport) Write() error {
	dir := filepath.Dir(c.filePath)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	data := []byte(fmt.Sprintf("%d", c.counter))
	return ioutil.WriteFile(c.filePath, data, 0644)
}
