package cache

import (
	"fmt"

	"github.com/go-git/go-billy/v5"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
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

// GetUserName returns the name the the user has used to configure git
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
	stdout1, err := identity.Fetch(c.repo, remote)
	if err != nil {
		return stdout1, err
	}

	stdout2, err := bug.Fetch(c.repo, remote)
	if err != nil {
		return stdout2, err
	}

	return stdout1 + stdout2, nil
}

// MergeAll will merge all the available remote bug and identities
func (c *RepoCache) MergeAll(remote string) <-chan entity.MergeResult {
	out := make(chan entity.MergeResult)

	// Intercept merge results to update the cache properly
	go func() {
		defer close(out)

		results := identity.MergeAll(c.repo, remote)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew, entity.MergeStatusUpdated:
				i := result.Entity.(*identity.Identity)
				c.muIdentity.Lock()
				c.identitiesExcerpts[result.Id] = NewIdentityExcerpt(i)
				c.muIdentity.Unlock()
			}
		}

		results = bug.MergeAll(c.repo, remote)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			switch result.Status {
			case entity.MergeStatusNew, entity.MergeStatusUpdated:
				b := result.Entity.(*bug.Bug)
				snap := b.Compile()
				c.muBug.Lock()
				c.bugExcerpts[result.Id] = NewBugExcerpt(b, &snap)
				c.muBug.Unlock()
			}
		}

		err := c.write()

		// No easy way out here ..
		if err != nil {
			panic(err)
		}
	}()

	return out
}

// Push update a remote with the local changes
func (c *RepoCache) Push(remote string) (string, error) {
	stdout1, err := identity.Push(c.repo, remote)
	if err != nil {
		return stdout1, err
	}

	stdout2, err := bug.Push(c.repo, remote)
	if err != nil {
		return stdout2, err
	}

	return stdout1 + stdout2, nil
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
	err := identity.SetUserIdentity(c.repo, i.Identity)
	if err != nil {
		return err
	}

	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	// Make sure that everything is fine
	if _, ok := c.identities[i.Id()]; !ok {
		panic("SetUserIdentity while the identity is not from the cache, something is wrong")
	}

	c.userIdentityId = i.Id()

	return nil
}

func (c *RepoCache) GetUserIdentity() (*IdentityCache, error) {
	if c.userIdentityId != "" {
		i, ok := c.identities[c.userIdentityId]
		if ok {
			return i, nil
		}
	}

	c.muIdentity.Lock()
	defer c.muIdentity.Unlock()

	i, err := identity.GetUserIdentity(c.repo)
	if err != nil {
		return nil, err
	}

	cached := NewIdentityCache(c, i)
	c.identities[i.Id()] = cached
	c.userIdentityId = i.Id()

	return cached, nil
}

func (c *RepoCache) GetUserIdentityExcerpt() (*IdentityExcerpt, error) {
	if c.userIdentityId == "" {
		id, err := identity.GetUserIdentityId(c.repo)
		if err != nil {
			return nil, err
		}
		c.userIdentityId = id
	}

	c.muIdentity.RLock()
	defer c.muIdentity.RUnlock()

	excerpt, ok := c.identitiesExcerpts[c.userIdentityId]
	if !ok {
		return nil, fmt.Errorf("cache: missing identity excerpt %v", c.userIdentityId)
	}
	return excerpt, nil
}

func (c *RepoCache) IsUserIdentitySet() (bool, error) {
	return identity.IsUserIdentitySet(c.repo)
}
