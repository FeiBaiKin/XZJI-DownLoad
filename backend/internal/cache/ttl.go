package cache

import (
	"sync"
	"time"
)

// TTLCache 是一个线程安全的通用 TTL 缓存
type TTLCache[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]*cacheItem[V]
	ttl      time.Duration
	stopCh   chan struct{}
	stopped  bool
}

type cacheItem[V any] struct {
	value     V
	expiresAt time.Time
}

// NewTTLCache 创建一个新的 TTL 缓存
func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
	c := &TTLCache[K, V]{
		items:  make(map[K]*cacheItem[V]),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
	// 启动后台清理 goroutine
	go c.cleanup()
	return c
}

// Get 获取缓存值，如果不存在或已过期则返回零值和 false
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	
	if time.Now().After(item.expiresAt) {
		var zero V
		return zero, false
	}
	
	return item.value, true
}

// Set 设置缓存值
func (c *TTLCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = &cacheItem[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete 删除缓存项
func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear 清空所有缓存
func (c *TTLCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*cacheItem[V])
}

// Stop 停止后台清理 goroutine
func (c *TTLCache[K, V]) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
	}
}

// cleanup 定期清理过期项
func (c *TTLCache[K, V]) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, item := range c.items {
				if now.After(item.expiresAt) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}
