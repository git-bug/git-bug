package cache

import (
	"fmt"

	"github.com/MichaelMure/git-bug/repository"
)

const lockfile = "lock"

type MultiRepoCache struct {
	repos map[string]*RepoCache
}

func NewMultiRepoCache() MultiRepoCache {
	return MultiRepoCache{
		repos: make(map[string]*RepoCache),
	}
}

// RegisterRepository register a named repository. Use this for multi-repo setup
func (c *MultiRepoCache) RegisterRepository(ref string, repo repository.ClockedRepo) error {
	r, err := NewRepoCache(repo)
	if err != nil {
		return err
	}

	c.repos[ref] = r
	return nil
}

// RegisterDefaultRepository register a unnamed repository. Use this for mono-repo setup
func (c *MultiRepoCache) RegisterDefaultRepository(repo repository.ClockedRepo) error {
	r, err := NewRepoCache(repo)
	if err != nil {
		return err
	}

	c.repos[""] = r
	return nil
}

// ResolveRepo retrieve a repository by name
func (c *MultiRepoCache) DefaultRepo() (*RepoCache, error) {
	if len(c.repos) != 1 {
		return nil, fmt.Errorf("repository is not unique")
	}

	for _, r := range c.repos {
		return r, nil
	}

	panic("unreachable")
}

// DefaultRepo retrieve the default repository
func (c *MultiRepoCache) ResolveRepo(ref string) (*RepoCache, error) {
	r, ok := c.repos[ref]
	if !ok {
		return nil, fmt.Errorf("unknown repo")
	}
	return r, nil
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
