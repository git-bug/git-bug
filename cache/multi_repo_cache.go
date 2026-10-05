package cache

import (
	"fmt"
)

const lockfile = "lock"

// MultiRepoCache is the root cache, holding multiple RepoCache by name. The
// empty name is that of an unnamed repository, as in a single-repo setup.
type MultiRepoCache struct {
	repos map[string]*RepoCache
}

func NewMultiRepoCache() *MultiRepoCache {
	return &MultiRepoCache{
		repos: make(map[string]*RepoCache),
	}
}

// Add registers a loaded repository under name, empty for an unnamed one.
func (c *MultiRepoCache) Add(name string, repo *RepoCache) {
	c.repos[name] = repo
}

// DefaultRepo retrieves the repository, and its name, if there is only one.
func (c *MultiRepoCache) DefaultRepo() (string, *RepoCache, error) {
	if len(c.repos) != 1 {
		return "", nil, fmt.Errorf("repository is not unique")
	}

	for name, r := range c.repos {
		return name, r, nil
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

// AllRepos returns all registered repositories, by name.
func (c *MultiRepoCache) AllRepos() map[string]*RepoCache {
	return c.repos
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
		r.registerAllObservers(nameFilter, observer)
	} else {
		if err := r.registerObserver(nameFilter, typename, observer); err != nil {
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
