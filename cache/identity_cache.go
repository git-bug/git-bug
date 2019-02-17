package cache

import (
	"github.com/MichaelMure/git-bug/identity"
)

// IdentityCache is a wrapper around an Identity. It provide multiple functions:
type IdentityCache struct {
	*identity.Identity
	repoCache *RepoCache
}

func NewIdentityCache(repoCache *RepoCache, id *identity.Identity) *IdentityCache {
	return &IdentityCache{
		Identity:  id,
		repoCache: repoCache,
	}
}

func (i *IdentityCache) Commit() error {
	return i.Identity.Commit(i.repoCache.repo)
}

func (i *IdentityCache) CommitAsNeeded() error {
	return i.Identity.CommitAsNeeded(i.repoCache.repo)
}
