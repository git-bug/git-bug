package cache

import (
	"fmt"

	"github.com/git-bug/git-bug/repository"
)

const lockfile = "lock"
const defaultRepoName = "__default"

// MultiRepoCache is the root cache, holding multiple RepoCache.
type MultiRepoCache struct {
	repos map[string]*RepoCache
}

func NewMultiRepoCache() *MultiRepoCache {
	return &MultiRepoCache{
		repos: make(map[string]*RepoCache),
	}
}

// RegisterRepository register a named repository. Use this for multi-repo setup
func (c *MultiRepoCache) RegisterRepository(repo repository.ClockedRepo, name string) (*RepoCache, chan BuildEvent) {
	r, events := NewNamedRepoCache(repo, name)

	// intercept events to make sure the cache building process succeeds properly
	out := make(chan BuildEvent)
	go func() {
		defer close(out)

		for event := range events {
			out <- event
			if event.Err != nil {
				return
			}
		}

		c.repos[name] = r
	}()

	return r, out
}

// RegisterDefaultRepository register an unnamed repository. Use this for single-repo setup
func (c *MultiRepoCache) RegisterDefaultRepository(repo repository.ClockedRepo) (*RepoCache, chan BuildEvent) {
	return c.RegisterRepository(repo, defaultRepoName)
}

// DefaultRepo retrieve the default repository
func (c *MultiRepoCache) DefaultRepo() (*RepoCache, error) {
	if len(c.repos) != 1 {
		return nil, fmt.Errorf("repository is not unique")
	}

	for _, r := range c.repos {
		return r, nil
	}

	panic("unreachable")
}

// ResolveRepo retrieve a repository by name
func (c *MultiRepoCache) ResolveRepo(name string) (*RepoCache, error) {
	r, ok := c.repos[name]
	if !ok {
		return nil, fmt.Errorf("unknown repo")
	}
	return r, nil
}

func (c *MultiRepoCache) RegisterObserver(observer Observer) {

}

func (c *MultiRepoCache) UnregisterObserver(observer Observer) {

}

// Close will do anything that is needed to close the cache properly
func (c *MultiRepoCache) Close() error {
	for _, cachedRepo := range c.repos {
		err := cachedRepo.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
