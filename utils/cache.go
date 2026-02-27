package utils

import (
	"sync"
	"time"
)

type CacheItem[V any] struct {
	Value      V
	Expiration int64
}

func (i CacheItem[V]) Expired() bool {
	return i.Expiration > 0 && time.Now().UnixMilli() > i.Expiration
}

type Cache[K comparable, V any] struct {
	mu         sync.RWMutex
	items      map[K]CacheItem[V]
	defaultTTL int64
}

func NewCache[K comparable, V any](defaultTTL time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		items:      make(map[K]CacheItem[V]),
		defaultTTL: defaultTTL.Milliseconds(),
	}
}

func (c *Cache[K, V]) Set(key K, value V, ttl ...time.Duration) {
	var exp int64
	now := time.Now().UnixMilli()

	if len(ttl) > 0 && ttl[0] > 0 {
		exp = now + ttl[0].Milliseconds()
	} else if c.defaultTTL > 0 {
		exp = now + c.defaultTTL
	}

	c.mu.Lock()
	c.items[key] = CacheItem[V]{Value: value, Expiration: exp}
	c.mu.Unlock()
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		var zero V
		return zero, false
	}

	if !item.Expired() {
		val := item.Value
		c.mu.RUnlock()
		return val, true
	}
	c.mu.RUnlock()

	// Item expired, need to delete it
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check since it might have been deleted/updated by another goroutine
	item, ok = c.items[key]
	if ok && item.Expired() {
		delete(c.items, key)
		var zero V
		return zero, false
	} else if ok {
		return item.Value, true
	}

	var zero V
	return zero, false
}

func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	c.items = make(map[K]CacheItem[V])
	c.mu.Unlock()
}
