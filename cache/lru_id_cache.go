package cache

import (
	"math"

	lru "github.com/hashicorp/golang-lru"

	"github.com/MichaelMure/git-bug/entity"
)

type lruIdCache struct {
	lru *lru.Cache
}

func newLRUIdCache() *lruIdCache {
	// we can ignore the error here as it would only fail if the size is negative.
	cache, _ := lru.New(math.MaxInt32)

	return &lruIdCache{
		cache,
	}
}

func (c *lruIdCache) Add(id entity.Id) bool {
	return c.lru.Add(id, nil)
}

func (c *lruIdCache) Contains(id entity.Id) bool {
	return c.lru.Contains(id)
}

func (c *lruIdCache) Get(id entity.Id) bool {
	_, present := c.lru.Get(id)
	return present
}

func (c *lruIdCache) GetOldest() (entity.Id, bool) {
	id, _, present := c.lru.GetOldest()
	return id.(entity.Id), present
}

func (c *lruIdCache) GetOldestToNewest() (ids []entity.Id) {
	interfaceKeys := c.lru.Keys()
	for _, id := range interfaceKeys {
		ids = append(ids, id.(entity.Id))
	}
	return
}

func (c *lruIdCache) Len() int {
	return c.lru.Len()
}

func (c *lruIdCache) Remove(id entity.Id) bool {
	return c.lru.Remove(id)
}
