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
		filePath: filePath,
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

func (c *PersistedLamport) Witness(time LamportTime) error {
	c.LamportClock.Witness(time)
	return c.Write()
}

func (c *PersistedLamport) Time() LamportTime {
	// Equivalent to:
	//
	// res = c.LamportClock.Time()
	// bugClock.Increment()
	//
	// ... but thread safe
	return c.Increment() - 1
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

	data := []byte(fmt.Sprintf("%d", c.LamportClock.Time()))
	return ioutil.WriteFile(c.filePath, data, 0644)
}
