package batch

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

// CacheMetricsCollector defines the interface for collecting cache metrics.
type CacheMetricsCollector interface {
	IncGets(n int64)
	IncSets(n int64)
	IncHits(n int64)
	IncMisses(n int64)
	IncEvictions(n int64)
	SetSize(n int64)
	GetMetrics() CacheMetrics
}

// AtomicCacheMetricsCollector is the default atomic-based metrics implementation.
type AtomicCacheMetricsCollector struct {
	gets      int64
	sets      int64
	hits      int64
	misses    int64
	evictions int64
	size      int64
}

func (m *AtomicCacheMetricsCollector) IncGets(n int64)      { atomic.AddInt64(&m.gets, n) }
func (m *AtomicCacheMetricsCollector) IncSets(n int64)      { atomic.AddInt64(&m.sets, n) }
func (m *AtomicCacheMetricsCollector) IncHits(n int64)      { atomic.AddInt64(&m.hits, n) }
func (m *AtomicCacheMetricsCollector) IncMisses(n int64)    { atomic.AddInt64(&m.misses, n) }
func (m *AtomicCacheMetricsCollector) IncEvictions(n int64) { atomic.AddInt64(&m.evictions, n) }
func (m *AtomicCacheMetricsCollector) SetSize(n int64)      { atomic.StoreInt64(&m.size, n) }
func (m *AtomicCacheMetricsCollector) GetMetrics() CacheMetrics {
	return CacheMetrics{
		Gets:      atomic.LoadInt64(&m.gets),
		Sets:      atomic.LoadInt64(&m.sets),
		Hits:      atomic.LoadInt64(&m.hits),
		Misses:    atomic.LoadInt64(&m.misses),
		Evictions: atomic.LoadInt64(&m.evictions),
		Size:      atomic.LoadInt64(&m.size),
	}
}

// PrometheusCacheMetricsCollector implements CacheMetricsCollector using prometheus metrics.
// (Requires github.com/prometheus/client_golang/prometheus)
type PrometheusCacheMetricsCollector struct {
	Gets      prometheus.Counter
	Sets      prometheus.Counter
	Hits      prometheus.Counter
	Misses    prometheus.Counter
	Evictions prometheus.Counter
	Size      prometheus.Gauge
}

func (m *PrometheusCacheMetricsCollector) IncGets(n int64)      { m.Gets.Add(float64(n)) }
func (m *PrometheusCacheMetricsCollector) IncSets(n int64)      { m.Sets.Add(float64(n)) }
func (m *PrometheusCacheMetricsCollector) IncHits(n int64)      { m.Hits.Add(float64(n)) }
func (m *PrometheusCacheMetricsCollector) IncMisses(n int64)    { m.Misses.Add(float64(n)) }
func (m *PrometheusCacheMetricsCollector) IncEvictions(n int64) { m.Evictions.Add(float64(n)) }
func (m *PrometheusCacheMetricsCollector) SetSize(n int64)      { m.Size.Set(float64(n)) }
func (m *PrometheusCacheMetricsCollector) GetMetrics() CacheMetrics {
	// Prometheus metrics are scraped via /metrics endpoint. This method returns zeros.
	return CacheMetrics{}
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
	metrics CacheMetricsCollector
}

func NewCache[K comparable, V any](metrics ...CacheMetricsCollector) *Cache[K, V] {
	var m CacheMetricsCollector
	if len(metrics) > 0 && metrics[0] != nil {
		m = metrics[0]
	} else {
		m = &AtomicCacheMetricsCollector{}
	}
	return &Cache[K, V]{
		data:    make(map[K]*value[V]),
		metrics: m,
	}
}

func (c *Cache[K, V]) StoreMany(ctx context.Context, kv map[K]V, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.IncSets(int64(len(kv)))

	for k, v := range kv {
		c.data[k] = newValue(v, ttl)
	}

	c.metrics.SetSize(int64(len(c.data)))
	return nil
}

func (c *Cache[K, V]) LoadMany(ctx context.Context, ks ...K) (map[K]V, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.metrics.IncGets(int64(len(ks)))

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

	c.metrics.IncHits(hits)
	c.metrics.IncMisses(misses)
	c.metrics.IncEvictions(evicted)

	return m, nil
}

// Metrics returns current cache metrics.
func (c *Cache[K, V]) Metrics() CacheMetrics {
	return c.metrics.GetMetrics()
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.data)
	c.metrics.SetSize(0)
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
		c.metrics.IncEvictions(int64(count))
		c.metrics.SetSize(int64(len(c.data)))
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
