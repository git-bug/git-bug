package cache

import (
	"fmt"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

type RepoCacheIdentity struct {
	*SubCache[*identity.Identity, *IdentityExcerpt, *IdentityCache]
}

func NewRepoCacheIdentity(repo repository.ClockedRepo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheIdentity {

	makeCached := func(i *identity.Identity, entityUpdated func(id entity.Id) error) *IdentityCache {
		return NewIdentityCache(i, repo, entityUpdated)
	}

	makeExcerpt := func(i *identity.Identity) *IdentityExcerpt {
		return NewIdentityExcerpt(i)
	}

	makeIndex := func(i *IdentityCache) []string {
		// no indexing
		return nil
	}

	sc := NewSubCache[*identity.Identity, *IdentityExcerpt, *IdentityCache](
		repo, resolvers, getUserIdentity,
		makeCached, makeExcerpt, makeIndex,
		"identity", "identities",
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheIdentity{SubCache: sc}
}

// ResolveIdentityImmutableMetadata retrieve an Identity that has the exact given metadata on
// one of its version. If multiple version have the same key, the first defined take precedence.
func (c *RepoCacheIdentity) ResolveIdentityImmutableMetadata(key string, value string) (*IdentityCache, error) {
	return c.ResolveMatcher(func(excerpt *IdentityExcerpt) bool {
		return excerpt.ImmutableMetadata[key] == value
	})
}

func (c *RepoCacheIdentity) NewIdentityFromGitUser() (*IdentityCache, error) {
	return c.NewIdentityFromGitUserRaw(nil)
}

func (c *RepoCacheIdentity) NewIdentityFromGitUserRaw(metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewFromGitUser(c.repo)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

// NewIdentity create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCacheIdentity) NewIdentity(name string, email string) (*IdentityCache, error) {
	return c.NewIdentityRaw(name, email, "", "", nil, nil)
}

// NewIdentityFull create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCacheIdentity) NewIdentityFull(name string, email string, login string, avatarUrl string, keys []*identity.Key) (*IdentityCache, error) {
	return c.NewIdentityRaw(name, email, login, avatarUrl, keys, nil)
}

func (c *RepoCacheIdentity) NewIdentityRaw(name string, email string, login string, avatarUrl string, keys []*identity.Key, metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewIdentityFull(c.repo, name, email, login, avatarUrl, keys)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

func (c *RepoCacheIdentity) finishIdentity(i *identity.Identity, metadata map[string]string) (*IdentityCache, error) {
	for key, value := range metadata {
		i.SetMetadata(key, value)
	}

	err := i.Commit(c.repo)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if _, has := c.cached[i.Id()]; has {
		return nil, fmt.Errorf("identity %s already exist in the cache", i.Id())
	}

	cached := NewIdentityCache(i, c.repo, c.entityUpdated)
	c.cached[i.Id()] = cached
	c.mu.Unlock()

	// force the write of the excerpt
	err = c.entityUpdated(i.Id())
	if err != nil {
		return nil, err
	}

	return cached, nil
}
