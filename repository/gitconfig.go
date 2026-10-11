package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/git-bug/gitconfig"
)

// configLockTimeout is how long an update waits for the lock of a config file
// held by another writer, git or git-bug. Writers hold it for a few syscalls,
// so running out means the lock was most likely left behind by a crash.
var configLockTimeout = time.Second

var _ Config = &gitConfig{}

// gitConfig gives access to the git configuration of a repository. Every read
// reads the files again, and every update is done under git's lock.
type gitConfig struct {
	env gitconfig.Env
}

func (c *gitConfig) Read() (*gitconfig.Config, error) {
	return c.env.Load()
}

func (c *gitConfig) Update(fn func(f *gitconfig.File) error) error {
	deadline := time.Now().Add(configLockTimeout)
	wait := time.Millisecond
	for {
		err := c.env.UpdateScope(gitconfig.ScopeLocal, fn)
		if !errors.Is(err, gitconfig.ErrLocked) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: another process is updating the config, or crashed doing so; if no git or git-bug process is running, remove the lock file", err)
		}
		time.Sleep(wait)
		wait = min(2*wait, 50*time.Millisecond)
	}
}
