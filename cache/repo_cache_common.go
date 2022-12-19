package cache

import (
	"github.com/go-git/go-billy/v5"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/entities/bug"
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

		author, err := c.GetUserIdentity()
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
		}

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

		results = bug.MergeAll(c.repo, c.resolvers, remote, author)
		for result := range results {
			out <- result

			if result.Err != nil {
				continue
			}

			// TODO: have subcache do the merging?
			switch result.Status {
			case entity.MergeStatusNew:
				b := result.Entity.(*bug.Bug)
				_, err := c.bugs.add(b)
			case entity.MergeStatusUpdated:
				_, err := c.bugs.entityUpdated(b)
				snap := b.Compile()
				c.muBug.Lock()
				c.bugExcerpts[result.Id] = NewBugExcerpt(b, snap)
				c.muBug.Unlock()
			}
		}

		err = c.write()
		if err != nil {
			out <- entity.NewMergeError(err, "")
			return
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
