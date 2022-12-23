package cache

import (
	"sync"

	"github.com/go-git/go-billy/v5"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

func (c *RepoCache) Name() string {
	return c.name
}

// LocalConfig give access to the repository scoped configuration
func (c *RepoCache) LocalConfig() repository.Config {
	return c.repo.LocalConfig()
}

// GlobalConfig give access to the global scoped configuration
func (c *RepoCache) GlobalConfig() repository.Config {
	return c.repo.GlobalConfig()
}

// AnyConfig give access to a merged local/global configuration
func (c *RepoCache) AnyConfig() repository.ConfigRead {
	return c.repo.AnyConfig()
}

// Keyring give access to a user-wide storage for secrets
func (c *RepoCache) Keyring() repository.Keyring {
	return c.repo.Keyring()
}

// GetUserName returns the name the user has used to configure git
func (c *RepoCache) GetUserName() (string, error) {
	return c.repo.GetUserName()
}

// GetUserEmail returns the email address that the user has used to configure git.
func (c *RepoCache) GetUserEmail() (string, error) {
	return c.repo.GetUserEmail()
}

// GetCoreEditor returns the name of the editor that the user has used to configure git.
func (c *RepoCache) GetCoreEditor() (string, error) {
	return c.repo.GetCoreEditor()
}

// GetRemotes returns the configured remotes repositories.
func (c *RepoCache) GetRemotes() (map[string]string, error) {
	return c.repo.GetRemotes()
}

// LocalStorage return a billy.Filesystem giving access to $RepoPath/.git/git-bug
func (c *RepoCache) LocalStorage() billy.Filesystem {
	return c.repo.LocalStorage()
}

// ReadData will attempt to read arbitrary data from the given hash
func (c *RepoCache) ReadData(hash repository.Hash) ([]byte, error) {
	return c.repo.ReadData(hash)
}

// StoreData will store arbitrary data and return the corresponding hash
func (c *RepoCache) StoreData(data []byte) (repository.Hash, error) {
	return c.repo.StoreData(data)
}

// Fetch retrieve updates from a remote
// This does not change the local bugs or identities state
func (c *RepoCache) Fetch(remote string) (string, error) {
	prefixes := make([]string, len(c.subcaches))
	for i, subcache := range c.subcaches {
		prefixes[i] = subcache.GetNamespace()
	}

	// fetch everything at once, to have a single auth step if required.
	return c.repo.FetchRefs(remote, prefixes...)
}

// MergeAll will merge all the available remote bug and identities
func (c *RepoCache) MergeAll(remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	dependency := [][]cacheMgmt{
		{c.identities},
		{c.bugs},
	}

	// run MergeAll according to entities dependencies and merge the results
	go func() {
		defer close(out)

		for _, subcaches := range dependency {
			var wg sync.WaitGroup
			for _, subcache := range subcaches {
				wg.Add(1)
				go func(subcache cacheMgmt) {
					for res := range subcache.MergeAll(remote) {
						out <- res
					}
					wg.Done()
				}(subcache)
			}
			wg.Wait()
		}
	}()

	return out
}

// Push update a remote with the local changes
func (c *RepoCache) Push(remote string) (string, error) {
	prefixes := make([]string, len(c.subcaches))
	for i, subcache := range c.subcaches {
		prefixes[i] = subcache.GetNamespace()
	}

	// push everything at once, to have a single auth step if required
	return c.repo.PushRefs(remote, prefixes...)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
func (c *RepoCache) Pull(remote string) error {
	_, err := c.Fetch(remote)
	if err != nil {
		return err
	}

	for merge := range c.MergeAll(remote) {
		if merge.Err != nil {
			return merge.Err
		}
		if merge.Status == entity.MergeStatusInvalid {
			return errors.Errorf("merge failure: %s", merge.Reason)
		}
	}

	return nil
}

func (c *RepoCache) SetUserIdentity(i *IdentityCache) error {
	c.muUserIdentity.RLock()
	defer c.muUserIdentity.RUnlock()

	// Make sure that everything is fine
	if _, err := c.identities.Resolve(i.Id()); err != nil {
		panic("SetUserIdentity while the identity is not from the cache, something is wrong")
	}

	err := identity.SetUserIdentity(c.repo, i.Identity)
	if err != nil {
		return err
	}

	c.userIdentityId = i.Id()

	return nil
}

func (c *RepoCache) GetUserIdentity() (*IdentityCache, error) {
	c.muUserIdentity.RLock()
	if c.userIdentityId != "" {
		defer c.muUserIdentity.RUnlock()
		return c.identities.Resolve(c.userIdentityId)
	}
	c.muUserIdentity.RUnlock()

	c.muUserIdentity.Lock()
	defer c.muUserIdentity.Unlock()

	i, err := identity.GetUserIdentityId(c.repo)
	if err != nil {
		return nil, err
	}

	c.userIdentityId = i

	return c.identities.Resolve(i)
}

func (c *RepoCache) GetUserIdentityExcerpt() (*IdentityExcerpt, error) {
	c.muUserIdentity.RLock()
	if c.userIdentityId != "" {
		defer c.muUserIdentity.RUnlock()
		return c.identities.ResolveExcerpt(c.userIdentityId)
	}
	c.muUserIdentity.RUnlock()

	c.muUserIdentity.Lock()
	defer c.muUserIdentity.Unlock()

	i, err := identity.GetUserIdentityId(c.repo)
	if err != nil {
		return nil, err
	}

	c.userIdentityId = i

	return c.identities.ResolveExcerpt(i)
}

func (c *RepoCache) IsUserIdentitySet() (bool, error) {
	return identity.IsUserIdentitySet(c.repo)
}
