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

// RegisterRepository registers a named repository. Use this for multi-repo setup
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

// RegisterDefaultRepository registers an unnamed repository. Use this for single-repo setup
func (c *MultiRepoCache) RegisterDefaultRepository(repo repository.ClockedRepo) (*RepoCache, chan BuildEvent) {
	return c.RegisterRepository(repo, defaultRepoName)
}

// DefaultRepo retrieves the default repository
func (c *MultiRepoCache) DefaultRepo() (*RepoCache, error) {
	if len(c.repos) != 1 {
		return nil, fmt.Errorf("repository is not unique")
	}

	for _, r := range c.repos {
		return r, nil
	}

	panic("unreachable")
}

// ResolveRepo retrieves a repository by name
func (c *MultiRepoCache) ResolveRepo(name string) (*RepoCache, error) {
	r, ok := c.repos[name]
	if !ok {
		return nil, fmt.Errorf("unknown repo")
	}
	return r, nil
}

// RegisterObserver registers an Observer on repo and entity, according to nameFilter and typename.
// - if nameFilter is empty, the observer is registered on all available repo
// - if nameFilter is not empty, the observer is registered on the repo with the matching name
// - if typename is empty, the observer is registered on all available entities
// - if typename is not empty, the observer is registered on the matching entity type only
func (c *MultiRepoCache) RegisterObserver(observer Observer, nameFilter string, typename string) error {
	if nameFilter == "" {
		for repoName, repo := range c.repos {
			if typename == "" {
				repo.registerAllObservers(repoName, observer)
			} else {
				if err := repo.registerObserver(repoName, typename, observer); err != nil {
					return err
				}
			}
		}
		return nil
	}

	r, err := c.ResolveRepo(nameFilter)
	if err != nil {
		return err
	}
	if typename == "" {
		r.registerAllObservers(r.Name(), observer)
	} else {
		if err := r.registerObserver(r.Name(), typename, observer); err != nil {
			return err
		}
	}
	return nil
}

// UnregisterObserver deregisters the observer from all repos and all entity types.
func (c *MultiRepoCache) UnregisterObserver(observer Observer) {
	for _, repo := range c.repos {
		repo.unregisterAllObservers(observer)
	}
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
