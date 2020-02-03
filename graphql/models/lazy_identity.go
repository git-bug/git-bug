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

type IdentityWrapper interface {
	Id() entity.Id
	Name() string
	Email() (string, error)
	AvatarUrl() (string, error)
	Keys() ([]*identity.Key, error)
	ValidKeysAtTime(time lamport.Time) ([]*identity.Key, error)
	DisplayName() string
	IsProtected() (bool, error)
	LastModificationLamport() (lamport.Time, error)
	LastModification() (timestamp.Timestamp, error)
}

var _ IdentityWrapper = &LazyIdentity{}

type LazyIdentity struct {
	cache   *cache.RepoCache
	excerpt *cache.IdentityExcerpt

	mu sync.Mutex
	id *cache.IdentityCache
}

func NewLazyIdentity(cache *cache.RepoCache, excerpt *cache.IdentityExcerpt) *LazyIdentity {
	return &LazyIdentity{
		cache:   cache,
		excerpt: excerpt,
	}
}

func (li *LazyIdentity) load() (*cache.IdentityCache, error) {
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

func (li *LazyIdentity) Id() entity.Id {
	return li.excerpt.Id
}

func (li *LazyIdentity) Name() string {
	return li.excerpt.Name
}

func (li *LazyIdentity) Email() (string, error) {
	id, err := li.load()
	if err != nil {
		return "", err
	}
	return id.Email(), nil
}

func (li *LazyIdentity) AvatarUrl() (string, error) {
	id, err := li.load()
	if err != nil {
		return "", err
	}
	return id.AvatarUrl(), nil
}

func (li *LazyIdentity) Keys() ([]*identity.Key, error) {
	id, err := li.load()
	if err != nil {
		return nil, err
	}
	return id.Keys(), nil
}

func (li *LazyIdentity) ValidKeysAtTime(time lamport.Time) ([]*identity.Key, error) {
	id, err := li.load()
	if err != nil {
		return nil, err
	}
	return id.ValidKeysAtTime(time), nil
}

func (li *LazyIdentity) DisplayName() string {
	return li.excerpt.DisplayName()
}

func (li *LazyIdentity) IsProtected() (bool, error) {
	id, err := li.load()
	if err != nil {
		return false, err
	}
	return id.IsProtected(), nil
}

func (li *LazyIdentity) LastModificationLamport() (lamport.Time, error) {
	id, err := li.load()
	if err != nil {
		return 0, err
	}
	return id.LastModificationLamport(), nil
}

func (li *LazyIdentity) LastModification() (timestamp.Timestamp, error) {
	id, err := li.load()
	if err != nil {
		return 0, err
	}
	return id.LastModification(), nil
}

var _ IdentityWrapper = &LoadedIdentity{}

type LoadedIdentity struct {
	identity.Interface
}

func NewLoadedIdentity(id identity.Interface) *LoadedIdentity {
	return &LoadedIdentity{Interface: id}
}

func (l LoadedIdentity) Email() (string, error) {
	return l.Interface.Email(), nil
}

func (l LoadedIdentity) AvatarUrl() (string, error) {
	return l.Interface.AvatarUrl(), nil
}

func (l LoadedIdentity) Keys() ([]*identity.Key, error) {
	return l.Interface.Keys(), nil
}

func (l LoadedIdentity) ValidKeysAtTime(time lamport.Time) ([]*identity.Key, error) {
	return l.Interface.ValidKeysAtTime(time), nil
}

func (l LoadedIdentity) IsProtected() (bool, error) {
	return l.Interface.IsProtected(), nil
}

func (l LoadedIdentity) LastModificationLamport() (lamport.Time, error) {
	return l.Interface.LastModificationLamport(), nil
}

func (l LoadedIdentity) LastModification() (timestamp.Timestamp, error) {
	return l.Interface.LastModification(), nil
}
