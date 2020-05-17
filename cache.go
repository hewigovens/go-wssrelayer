package wsrelayer

import (
	"time"

	"github.com/cloudflare/golibs/lrucache"
)

type CacheItem struct {
	Id         string
	OriginalId string
	Chan       chan []byte
}

type RequestCache struct {
	cache *lrucache.LRUCache
}

func NewRequestCache() *RequestCache {
	return &RequestCache{
		cache: lrucache.NewLRUCache(100),
	}
}

func (c *RequestCache) Get(id string) *CacheItem {
	hit, ok := c.cache.Get(id)
	if !ok {
		return nil
	}
	item, ok := hit.(CacheItem)
	if !ok {
		return nil
	}
	return &item
}

func (c *RequestCache) Set(id string, value interface{}, timeout time.Time) {
	c.cache.Set(id, value, timeout)
}

func (c *RequestCache) Del(id string) {
	_, _ = c.cache.Del(id)
}
