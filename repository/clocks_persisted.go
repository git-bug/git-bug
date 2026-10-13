package repository

import (
	"errors"
	"os"
	"sync"

	"github.com/git-bug/git-bug/util/lamport"
)

const clockPath = "clocks"

var _ RepoClock = &persistedClocks{}

// persistedClocks are Lamport clocks persisted as files in a LocalStorage, in
// clocks/.
type persistedClocks struct {
	storage LocalStorage

	mu     sync.Mutex
	clocks map[string]lamport.Clock
}

// NewPersistedClocks returns the Lamport clocks persisted in storage.
func NewPersistedClocks(storage LocalStorage) RepoClock {
	return &persistedClocks{
		storage: storage,
		clocks:  make(map[string]lamport.Clock),
	}
}

func (p *persistedClocks) AllClocks() (map[string]lamport.Clock, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]lamport.Clock)

	files, err := p.storage.ReadDir(clockPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		name := file.Name()
		if c, ok := p.clocks[name]; ok {
			result[name] = c
		} else {
			c, err := lamport.LoadPersistedClock(p.storage, p.storage.Join(clockPath, name))
			if errors.Is(err, lamport.ErrClockNotExist) {
				// an empty file: another process is creating it, or died
				// doing so. Not a clock yet.
				continue
			}
			if err != nil {
				return nil, err
			}
			p.clocks[name] = c
			result[name] = c
		}
	}

	return result, nil
}

// GetClock return a Lamport clock stored in the Repo, or ErrClockNotExist if
// there is none.
func (p *persistedClocks) GetClock(name string) (lamport.Clock, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.clocks[name]; ok {
		return c, nil
	}

	c, err := lamport.LoadPersistedClock(p.storage, p.storage.Join(clockPath, name))
	if err != nil {
		return nil, err
	}

	p.clocks[name] = c
	return c, nil
}

// GetOrCreateClock return a Lamport clock stored in the Repo, creating it at the
// initial time if there is none, or if the existing one is corrupted. See
// lamport.GetOrCreatePersistedClock for what the initial time must hold.
func (p *persistedClocks) GetOrCreateClock(name string, initial lamport.Time) (lamport.Clock, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// goes to the file even with a clock cached: it's the file that may be
	// missing or corrupted
	c, err := lamport.GetOrCreatePersistedClock(p.storage, p.storage.Join(clockPath, name), initial)
	if err != nil {
		return nil, err
	}

	if cached, ok := p.clocks[name]; ok {
		return cached, nil
	}

	p.clocks[name] = c
	return c, nil
}

// Increment is equivalent to c = GetClock(name) + c.Increment()
func (p *persistedClocks) Increment(name string) (lamport.Time, error) {
	c, err := p.GetClock(name)
	if err != nil {
		return lamport.Time(0), err
	}
	return c.Increment()
}

// Witness is equivalent to c = GetClock(name) + c.Witness(time)
func (p *persistedClocks) Witness(name string, time lamport.Time) error {
	c, err := p.GetClock(name)
	if err != nil {
		return err
	}
	return c.Witness(time)
}
