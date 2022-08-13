package cache

import (
	"github.com/MichaelMure/git-bug/entity"
)

func makeResolvers(cache *RepoCache) entity.Resolvers {
	return entity.Resolvers{
		&IdentityCache{}: newIdentityCacheResolver(cache),
		&BugCache{}:      newBugCacheResolver(cache),
	}
}

var _ entity.Resolver = &identityCacheResolver{}

// identityCacheResolver is an identity Resolver that retrieve identities from
// the cache
type identityCacheResolver struct {
	cache *RepoCache
}

func newIdentityCacheResolver(cache *RepoCache) *identityCacheResolver {
	return &identityCacheResolver{cache: cache}
}

func (i *identityCacheResolver) Resolve(id entity.Id) (entity.Interface, error) {
	return i.cache.ResolveIdentity(id)
}

var _ entity.Resolver = &bugCacheResolver{}

type bugCacheResolver struct {
	cache *RepoCache
}

func newBugCacheResolver(cache *RepoCache) *bugCacheResolver {
	return &bugCacheResolver{cache: cache}
}

func (b *bugCacheResolver) Resolve(id entity.Id) (entity.Interface, error) {
	return b.cache.ResolveBug(id)
}
