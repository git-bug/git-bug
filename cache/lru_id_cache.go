package cache

import (
	"math"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/MichaelMure/git-bug/entity"
)

type lruIdCache struct {
	*lru.Cache[entity.Id, *struct{}]
}

func newLRUIdCache() lruIdCache {
	// we can ignore the error here as it would only fail if the size is negative.
	cache, _ := lru.New[entity.Id, *struct{}](math.MaxInt32)
	return lruIdCache{Cache: cache}
}

func (c *lruIdCache) Add(id entity.Id) bool {
	return c.Cache.Add(id, nil)
}
func (c *lruIdCache) GetOldest() (entity.Id, bool) {
	id, _, present := c.Cache.GetOldest()
	return id, present
}

func (c *lruIdCache) GetOldestToNewest() (ids []entity.Id) {
	return c.Keys()
}
