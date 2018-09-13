package lamport

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Persisted struct {
	Clock
	filePath string
}

func NewPersisted(filePath string) (*Persisted, error) {
	clock := &Persisted{
		Clock:    NewClock(),
		filePath: filePath,
	}

	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return nil, err
	}

	return clock, nil
}

func LoadPersisted(filePath string) (*Persisted, error) {
	clock := &Persisted{
		filePath: filePath,
	}

	err := clock.read()
	if err != nil {
		return nil, err
	}

	return clock, nil
}

func (c *Persisted) Increment() (Time, error) {
	time := c.Clock.Increment()
	return time, c.Write()
}

func (c *Persisted) Witness(time Time) error {
	// TODO: rework so that we write only when the clock was actually updated
	c.Clock.Witness(time)
	return c.Write()
}

func (c *Persisted) read() error {
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

	c.Clock = NewClockWithTime(value)

	return nil
}

func (c *Persisted) Write() error {
	data := []byte(fmt.Sprintf("%d", c.counter))
	return ioutil.WriteFile(c.filePath, data, 0644)
}
