package repository

import (
	"bytes"
	"sync"

	"github.com/git-bug/gitconfig"
)

var _ Config = &memConfig{}

// memConfig is a configuration held in memory, as a single file read and
// edited as git does.
type memConfig struct {
	mu   sync.Mutex
	data []byte
}

func (c *memConfig) Read() (*gitconfig.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, err := gitconfig.Parse("memory", c.data)
	if err != nil {
		return nil, err
	}
	return f.Config(), nil
}

func (c *memConfig) Update(fn func(f *gitconfig.File) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, err := gitconfig.Parse("memory", c.data)
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		return err
	}
	c.data = bytes.Clone(f.Bytes())
	return nil
}
