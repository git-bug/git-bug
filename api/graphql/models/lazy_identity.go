package models

import (
	"context"
	"fmt"
	"sync"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
)

// IdentityWrapper is an interface used by the GraphQL resolvers to handle an identity.
// Depending on the situation, an Identity can already be fully loaded in memory or not.
// This interface is used to wrap either a lazyIdentity or a loadedIdentity depending on the situation.
type IdentityWrapper interface {
	Id() entity.Id
	Name() string
	Email(ctx context.Context) (string, error)
	Login(ctx context.Context) (string, error)
	AvatarUrl(ctx context.Context) (string, error)
	Keys(ctx context.Context) ([]*identity.Key, error)
	DisplayName() string
	IsProtected(ctx context.Context) (bool, error)

	// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
	IsEntity()
}

var _ IdentityWrapper = &lazyIdentity{}

type lazyIdentity struct {
	cache   *cache.RepoCache
	excerpt *cache.IdentityExcerpt

	mu sync.Mutex
	id *cache.IdentityCache
}

func NewLazyIdentity(cache *cache.RepoCache, excerpt *cache.IdentityExcerpt) *lazyIdentity {
	return &lazyIdentity{
		cache:   cache,
		excerpt: excerpt,
	}
}

func (li *lazyIdentity) load(ctx context.Context) (*cache.IdentityCache, error) {
	li.mu.Lock()
	defer li.mu.Unlock()

	if li.id != nil {
		return li.id, nil
	}

	id, err := li.cache.Identities().Resolve(ctx, li.excerpt.Id())
	if err != nil {
		return nil, fmt.Errorf("cache: resolve identity %v: %w", li.excerpt.Id(), err)
	}
	li.id = id
	return id, nil
}

func (li *lazyIdentity) Id() entity.Id {
	return li.excerpt.Id()
}

func (li *lazyIdentity) Name() string {
	return li.excerpt.Name
}

func (li *lazyIdentity) DisplayName() string {
	return li.excerpt.DisplayName()
}

func (li *lazyIdentity) Email(ctx context.Context) (string, error) {
	id, err := li.load(ctx)
	if err != nil {
		return "", err
	}
	return id.Email(), nil
}

func (li *lazyIdentity) Login(ctx context.Context) (string, error) {
	id, err := li.load(ctx)
	if err != nil {
		return "", err
	}
	return id.Login(), nil
}

func (li *lazyIdentity) AvatarUrl(ctx context.Context) (string, error) {
	id, err := li.load(ctx)
	if err != nil {
		return "", err
	}
	return id.AvatarUrl(), nil
}

func (li *lazyIdentity) Keys(ctx context.Context) ([]*identity.Key, error) {
	id, err := li.load(ctx)
	if err != nil {
		return nil, err
	}
	return id.Keys(), nil
}

func (li *lazyIdentity) IsProtected(ctx context.Context) (bool, error) {
	id, err := li.load(ctx)
	if err != nil {
		return false, err
	}
	return id.IsProtected(), nil
}

// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
func (li *lazyIdentity) IsEntity() {}

var _ IdentityWrapper = &loadedIdentity{}

type loadedIdentity struct {
	identity.Interface
}

func NewLoadedIdentity(id identity.Interface) *loadedIdentity {
	return &loadedIdentity{Interface: id}
}

func (l loadedIdentity) Email(ctx context.Context) (string, error) {
	return l.Interface.Email(), nil
}

func (l loadedIdentity) Login(ctx context.Context) (string, error) {
	return l.Interface.Login(), nil
}

func (l loadedIdentity) AvatarUrl(ctx context.Context) (string, error) {
	return l.Interface.AvatarUrl(), nil
}

func (l loadedIdentity) Keys(ctx context.Context) ([]*identity.Key, error) {
	return l.Interface.Keys(), nil
}

func (l loadedIdentity) IsProtected(ctx context.Context) (bool, error) {
	return l.Interface.IsProtected(), nil
}

// IsEntity is a sign post-method for gqlgen, to mark compliance to an interface.
func (l loadedIdentity) IsEntity() {}
