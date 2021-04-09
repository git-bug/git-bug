package cache

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

var _ identity.Resolver = &identityCacheResolver{}

// identityCacheResolver is an identity Resolver that retrieve identities from
// the cache
type identityCacheResolver struct {
	cache *RepoCache
}

func newIdentityCacheResolver(cache *RepoCache) *identityCacheResolver {
	return &identityCacheResolver{cache: cache}
}

func (i *identityCacheResolver) ResolveIdentity(id entity.Id) (identity.Interface, error) {
	return i.cache.ResolveIdentity(id)
}

var _ identity.Resolver = &identityCacheResolverNoLock{}

// identityCacheResolverNoLock is an identity Resolver that retrieve identities from
// the cache, without locking it.
type identityCacheResolverNoLock struct {
	cache *RepoCache
}

func newIdentityCacheResolverNoLock(cache *RepoCache) *identityCacheResolverNoLock {
	return &identityCacheResolverNoLock{cache: cache}
}

func (ir *identityCacheResolverNoLock) ResolveIdentity(id entity.Id) (identity.Interface, error) {
	cached, ok := ir.cache.identities[id]
	if ok {
		return cached, nil
	}

	i, err := identity.ReadLocal(ir.cache.repo, id)
	if err != nil {
		return nil, err
	}

	cached = NewIdentityCache(ir.cache, i)
	ir.cache.identities[id] = cached

	return cached, nil
}
