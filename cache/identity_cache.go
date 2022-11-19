package cache

import (
	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

var _ identity.Interface = &IdentityCache{}

// IdentityCache is a wrapper around an Identity for caching.
type IdentityCache struct {
	entityUpdated func(id entity.Id) error
	repo          repository.ClockedRepo

	*identity.Identity
}

func NewIdentityCache(subcache *RepoCacheIdentity, id *identity.Identity) *IdentityCache {
	return &IdentityCache{
		entityUpdated: subcache.entityUpdated,
		repo:          subcache.repo,
		Identity:      id,
	}
}

func (i *IdentityCache) notifyUpdated() error {
	return i.entityUpdated(i.Identity.Id())
}

func (i *IdentityCache) Mutate(repo repository.RepoClock, f func(*identity.Mutator)) error {
	err := i.Identity.Mutate(repo, f)
	if err != nil {
		return err
	}
	return i.notifyUpdated()
}

func (i *IdentityCache) Commit() error {
	err := i.Identity.Commit(i.repo)
	if err != nil {
		return err
	}
	return i.notifyUpdated()
}

func (i *IdentityCache) CommitAsNeeded() error {
	err := i.Identity.CommitAsNeeded(i.repo)
	if err != nil {
		return err
	}
	return i.notifyUpdated()
}
