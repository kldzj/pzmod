package steam

import "time"

type WorkshopCacheItem struct {
	WorkshopItem
	LastUpdated int64
}

type WorkshopItemCache struct {
	expiration time.Duration
	items      map[string]WorkshopCacheItem
}

func NewWorkshopItemCache(expiration time.Duration) *WorkshopItemCache {
	return &WorkshopItemCache{
		expiration: expiration,
		items:      make(map[string]WorkshopCacheItem),
	}
}

func (c *WorkshopItemCache) Get(id string) (*WorkshopItem, bool) {
	item, ok := c.items[id]
	if !ok {
		return nil, false
	}

	if time.Now().Unix()-item.LastUpdated > int64(c.expiration.Seconds()) {
		return nil, false
	}

	return &item.WorkshopItem, true
}

func (c *WorkshopItemCache) Set(id string, item WorkshopItem) {
	c.items[id] = WorkshopCacheItem{
		WorkshopItem: item,
		LastUpdated:  time.Now().Unix(),
	}
}

func (c *WorkshopItemCache) Delete(id string) {
	delete(c.items, id)
}

func (c *WorkshopItemCache) Clear() {
	c.items = make(map[string]WorkshopCacheItem)
}
