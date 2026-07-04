package steam

import (
	"sync"
	"time"
)

// Cache stores Workshop items to avoid refetching within a TTL.
type Cache interface {
	Get(id string) (WorkshopItem, bool)
	Set(id string, item WorkshopItem)
	Delete(id string)
	Clear()
}

type cacheEntry struct {
	item    WorkshopItem
	expires time.Time
}

// memCache is a goroutine-safe in-memory TTL cache with an injectable clock.
type memCache struct {
	mu  sync.Mutex
	ttl time.Duration
	now func() time.Time
	m   map[string]cacheEntry
}

// NewMemCache returns an in-memory cache with the given TTL. If now is nil,
// time.Now is used.
func NewMemCache(ttl time.Duration, now func() time.Time) Cache {
	if now == nil {
		now = time.Now
	}
	return &memCache{ttl: ttl, now: now, m: make(map[string]cacheEntry)}
}

func (c *memCache) Get(id string) (WorkshopItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[id]
	if !ok || c.now().After(e.expires) {
		return WorkshopItem{}, false
	}
	return e.item, true
}

func (c *memCache) Set(id string, item WorkshopItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[id] = cacheEntry{item: item, expires: c.now().Add(c.ttl)}
}

func (c *memCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, id)
}

func (c *memCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]cacheEntry)
}
