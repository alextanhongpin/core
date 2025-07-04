package batch

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// CacheMetrics contains performance metrics for the cache.
type CacheMetrics struct {
	Gets      int64 // Number of get operations
	Sets      int64 // Number of set operations
	Hits      int64 // Number of cache hits
	Misses    int64 // Number of cache misses
	Evictions int64 // Number of expired entries evicted
	Size      int64 // Current cache size
}

type cache[K comparable, V any] interface {
	LoadMany(ctx context.Context, keys ...K) (map[K]V, error)
	StoreMany(ctx context.Context, kv map[K]V, ttl time.Duration) error
	Metrics() CacheMetrics
	Clear()
}

var _ cache[int, any] = (*Cache[int, any])(nil)

type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	data    map[K]*value[V]
	metrics CacheMetrics
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		data: make(map[K]*value[V]),
	}
}

func (c *Cache[K, V]) StoreMany(ctx context.Context, kv map[K]V, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt64(&c.metrics.Sets, int64(len(kv)))

	for k, v := range kv {
		c.data[k] = newValue(v, ttl)
	}

	atomic.StoreInt64(&c.metrics.Size, int64(len(c.data)))
	return nil
}

func (c *Cache[K, V]) LoadMany(ctx context.Context, ks ...K) (map[K]V, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	atomic.AddInt64(&c.metrics.Gets, int64(len(ks)))

	m := make(map[K]V)
	hits, misses := int64(0), int64(0)
	evicted := int64(0)

	for _, k := range ks {
		v, ok := c.data[k]
		if !ok {
			misses++
			continue
		}
		if v.expired() {
			// Note: We don't delete here to avoid lock upgrade
			// Cleanup will happen in a separate method
			evicted++
			misses++
			continue
		}

		m[k] = v.data
		hits++
	}

	atomic.AddInt64(&c.metrics.Hits, hits)
	atomic.AddInt64(&c.metrics.Misses, misses)
	atomic.AddInt64(&c.metrics.Evictions, evicted)

	return m, nil
}

// Metrics returns current cache metrics.
func (c *Cache[K, V]) Metrics() CacheMetrics {
	return CacheMetrics{
		Gets:      atomic.LoadInt64(&c.metrics.Gets),
		Sets:      atomic.LoadInt64(&c.metrics.Sets),
		Hits:      atomic.LoadInt64(&c.metrics.Hits),
		Misses:    atomic.LoadInt64(&c.metrics.Misses),
		Evictions: atomic.LoadInt64(&c.metrics.Evictions),
		Size:      atomic.LoadInt64(&c.metrics.Size),
	}
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.data)
	atomic.StoreInt64(&c.metrics.Size, 0)
}

// CleanupExpired removes expired entries from the cache.
func (c *Cache[K, V]) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for k, v := range c.data {
		if v.expired() {
			delete(c.data, k)
			count++
		}
	}

	if count > 0 {
		atomic.AddInt64(&c.metrics.Evictions, int64(count))
		atomic.StoreInt64(&c.metrics.Size, int64(len(c.data)))
	}

	return count
}

type value[T any] struct {
	data     T
	deadline time.Time
}

func newValue[T any](v T, ttl time.Duration) *value[T] {
	return &value[T]{
		data:     v,
		deadline: time.Now().Add(ttl),
	}
}

func (v *value[T]) expired() bool {
	return gte(time.Now(), v.deadline)
}

func gte(a, b time.Time) bool {
	return !a.Before(b)
}
