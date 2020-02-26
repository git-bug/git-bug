package models

import (
	"fmt"
	"sync"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/lamport"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

// IdentityWrapper is an interface used by the GraphQL resolvers to handle an identity.
// Depending on the situation, an Identity can already be fully loaded in memory or not.
// This interface is used to wrap either a lazyIdentity or a loadedIdentity depending on the situation.
type IdentityWrapper interface {
	Id() entity.Id
	Name() string
	Email() (string, error)
	Login() (string, error)
	AvatarUrl() (string, error)
	Keys() ([]*identity.Key, error)
	ValidKeysAtTime(time lamport.Time) ([]*identity.Key, error)
	DisplayName() string
	IsProtected() (bool, error)
	LastModificationLamport() (lamport.Time, error)
	LastModification() (timestamp.Timestamp, error)
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

func (li *lazyIdentity) load() (*cache.IdentityCache, error) {
	if li.id != nil {
		return li.id, nil
	}

	li.mu.Lock()
	defer li.mu.Unlock()

	id, err := li.cache.ResolveIdentity(li.excerpt.Id)
	if err != nil {
		return nil, fmt.Errorf("cache: missing identity %v", li.excerpt.Id)
	}
	li.id = id
	return id, nil
}

func (li *lazyIdentity) Id() entity.Id {
	return li.excerpt.Id
}

func (li *lazyIdentity) Name() string {
	return li.excerpt.Name
}

func (li *lazyIdentity) Email() (string, error) {
	id, err := li.load()
	if err != nil {
		return "", err
	}
	return id.Email(), nil
}

func (li *lazyIdentity) Login() (string, error) {
	id, err := li.load()
	if err != nil {
		return "", err
	}
	return id.Login(), nil
}

func (li *lazyIdentity) AvatarUrl() (string, error) {
	id, err := li.load()
	if err != nil {
		return "", err
	}
	return id.AvatarUrl(), nil
}

func (li *lazyIdentity) Keys() ([]*identity.Key, error) {
	id, err := li.load()
	if err != nil {
		return nil, err
	}
	return id.Keys(), nil
}

func (li *lazyIdentity) ValidKeysAtTime(time lamport.Time) ([]*identity.Key, error) {
	id, err := li.load()
	if err != nil {
		return nil, err
	}
	return id.ValidKeysAtTime(time), nil
}

func (li *lazyIdentity) DisplayName() string {
	return li.excerpt.DisplayName()
}

func (li *lazyIdentity) IsProtected() (bool, error) {
	id, err := li.load()
	if err != nil {
		return false, err
	}
	return id.IsProtected(), nil
}

func (li *lazyIdentity) LastModificationLamport() (lamport.Time, error) {
	id, err := li.load()
	if err != nil {
		return 0, err
	}
	return id.LastModificationLamport(), nil
}

func (li *lazyIdentity) LastModification() (timestamp.Timestamp, error) {
	id, err := li.load()
	if err != nil {
		return 0, err
	}
	return id.LastModification(), nil
}

var _ IdentityWrapper = &loadedIdentity{}

type loadedIdentity struct {
	identity.Interface
}

func NewLoadedIdentity(id identity.Interface) *loadedIdentity {
	return &loadedIdentity{Interface: id}
}

func (l loadedIdentity) Email() (string, error) {
	return l.Interface.Email(), nil
}

func (l loadedIdentity) Login() (string, error) {
	return l.Interface.Login(), nil
}

func (l loadedIdentity) AvatarUrl() (string, error) {
	return l.Interface.AvatarUrl(), nil
}

func (l loadedIdentity) Keys() ([]*identity.Key, error) {
	return l.Interface.Keys(), nil
}

func (l loadedIdentity) ValidKeysAtTime(time lamport.Time) ([]*identity.Key, error) {
	return l.Interface.ValidKeysAtTime(time), nil
}

func (l loadedIdentity) IsProtected() (bool, error) {
	return l.Interface.IsProtected(), nil
}

func (l loadedIdentity) LastModificationLamport() (lamport.Time, error) {
	return l.Interface.LastModificationLamport(), nil
}

func (l loadedIdentity) LastModification() (timestamp.Timestamp, error) {
	return l.Interface.LastModification(), nil
}
