package repository

import (
	"strconv"
	"time"

	"github.com/git-bug/gitconfig"
)

// Config gives access to the git configuration of a repository, as `git config`
// does: reads see every scope merged, writes go to the repository's config file.
type Config interface {
	// Read reads the effective configuration as it is now: every scope merged
	// as git does, the last value of a key winning. The result is an immutable
	// snapshot: later updates don't change it.
	Read() (*gitconfig.Config, error)

	// Update edits the repository's config file with fn, atomically: if fn
	// returns an error, nothing is written, otherwise all its edits are.
	// Concurrent updates, from git or git-bug, never lose each other's edits.
	// fn must not keep the File after it returns.
	Update(fn func(f *gitconfig.File) error) error
}

func ParseTimestamp(s string) (time.Time, error) {
	timestamp, err := strconv.Atoi(s)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(int64(timestamp), 0), nil
}
