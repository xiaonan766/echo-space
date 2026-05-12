package cache

import (
	"sync"
	"time"
)

type LocalCache struct {
	mu    sync.RWMutex
	items map[string]localCacheItem
}

type localCacheItem struct {
	value    []byte
	expireAt time.Time
}

func NewLocalCache() *LocalCache {
	return &LocalCache{
		items: make(map[string]localCacheItem),
	}
}

func (c *LocalCache) Set(key string, value []byte, ttl time.Duration) {
	valueCopy := append([]byte(nil), value...)

	var expireAt time.Time
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = localCacheItem{
		value:    valueCopy,
		expireAt: expireAt,
	}
}

func (c *LocalCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if item.expired(time.Now()) {
		c.Delete(key)
		return nil, false
	}

	return append([]byte(nil), item.value...), true
}

func (c *LocalCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (i localCacheItem) expired(now time.Time) bool {
	return !i.expireAt.IsZero() && now.After(i.expireAt)
}
