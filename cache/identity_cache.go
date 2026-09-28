package cache

import (
	"sync"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/repository"
)

var _ identity.Interface = &IdentityCache{}
var _ CacheEntity = &IdentityCache{}

// IdentityCache is a wrapper around an Identity for caching.
type IdentityCache struct {
	repo     repository.ClockedRepo
	onCommit func() error // called after each commit

	mu sync.Mutex
	*identity.Identity
}

func NewIdentityCache(i *identity.Identity, repo repository.ClockedRepo, onCommit func() error) *IdentityCache {
	return &IdentityCache{
		repo:     repo,
		onCommit: onCommit,
		Identity: i,
	}
}

func (i *IdentityCache) Mutate(repo repository.RepoClock, f func(*identity.Mutator)) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.Identity.Mutate(repo, f)
}

func (i *IdentityCache) Commit() error {
	i.mu.Lock()
	err := i.Identity.Commit(i.repo)
	i.mu.Unlock()
	if err != nil {
		return err
	}
	return i.onCommit()
}

func (i *IdentityCache) CommitAsNeeded() error {
	i.mu.Lock()
	if !i.Identity.NeedCommit() {
		i.mu.Unlock()
		return nil
	}
	err := i.Identity.Commit(i.repo)
	i.mu.Unlock()
	if err != nil {
		return err
	}
	return i.onCommit()
}
