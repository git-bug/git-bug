package cache

import (
	lru "github.com/hashicorp/golang-lru"

	"github.com/MichaelMure/git-bug/entity"
)

type LRUIdCache struct {
	parentCache *lru.Cache
	maxSize     int
	onEvict     func(id entity.Id)
}

func NewLRUIdCache(size int, onEvicted func(id entity.Id)) (*LRUIdCache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}

	return &LRUIdCache{
		cache,
		size,
		onEvicted,
	}, nil
}

func (c *LRUIdCache) Add(id entity.Id) bool {
	return c.parentCache.Add(id, nil)
}

func (c *LRUIdCache) Contains(id entity.Id) bool {
	return c.parentCache.Contains(id)
}

func (c *LRUIdCache) Get(id entity.Id) bool {
	_, present := c.parentCache.Get(id)
	return present
}

func (c *LRUIdCache) GetOldest() (entity.Id, bool) {
	id, _, present := c.parentCache.GetOldest()
	return id.(entity.Id), present
}

func (c *LRUIdCache) GetAll() (ids []entity.Id) {
	interfaceKeys := c.parentCache.Keys()
	for _, id := range interfaceKeys {
		ids = append(ids, id.(entity.Id))
	}
	return
}

func (c *LRUIdCache) Len() int {
	return c.parentCache.Len()
}

func (c *LRUIdCache) Remove(id entity.Id) bool {
	return c.parentCache.Remove(id)
}

func (c *LRUIdCache) Resize(size int) int {
	return c.parentCache.Resize(size)
}
