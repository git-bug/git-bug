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
