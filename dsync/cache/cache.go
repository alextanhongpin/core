// Package cache provides a Redis-based cache implementation with atomic operations.
package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	redis "github.com/redis/go-redis/v9"
)

// ErrNotExist is returned when a key does not exist in the cache.
// ErrExists is returned when trying to store a key that already exists (StoreOnce).
// ErrValueMismatch is returned when compare operations fail due to value differences.
// ErrUnexpectedType is returned when Redis returns an unexpected data type.
var (
	ErrNotExist       = redis.Nil
	ErrExists         = errors.New("key already exists")
	ErrValueMismatch  = errors.New("compare operation failed: value mismatch")
	ErrUnexpectedType = errors.New("unexpected value type from Redis")
)

// MetricsCollector defines the interface for collecting cache metrics.
type MetricsCollector interface {
	IncTotalRequests()
	IncHits()
	IncMisses()
	IncSets()
	IncDeletes()
}

type AtomicCacheMetrics struct {
	totalRequests int64
	hits          int64
	misses        int64
	sets          int64
	deletes       int64
}

func (m *AtomicCacheMetrics) IncTotalRequests() { atomic.AddInt64(&m.totalRequests, 1) }
func (m *AtomicCacheMetrics) IncHits()          { atomic.AddInt64(&m.hits, 1) }
func (m *AtomicCacheMetrics) IncMisses()        { atomic.AddInt64(&m.misses, 1) }
func (m *AtomicCacheMetrics) IncSets()          { atomic.AddInt64(&m.sets, 1) }
func (m *AtomicCacheMetrics) IncDeletes()       { atomic.AddInt64(&m.deletes, 1) }

// PrometheusCacheMetrics implements MetricsCollector using prometheus metrics.
type PrometheusCacheMetrics struct {
	TotalRequests prometheus.Counter
	Hits          prometheus.Counter
	Misses        prometheus.Counter
	Sets          prometheus.Counter
	Deletes       prometheus.Counter
}

func (m *PrometheusCacheMetrics) IncTotalRequests() { m.TotalRequests.Inc() }
func (m *PrometheusCacheMetrics) IncHits()          { m.Hits.Inc() }
func (m *PrometheusCacheMetrics) IncMisses()        { m.Misses.Inc() }
func (m *PrometheusCacheMetrics) IncSets()          { m.Sets.Inc() }
func (m *PrometheusCacheMetrics) IncDeletes()       { m.Deletes.Inc() }

// Cacheable defines the interface for cache operations with atomic guarantees.
// All operations are thread-safe and provide strong consistency through Redis.
type Cacheable interface {
	// CompareAndDelete atomically deletes a key only if its current value matches the expected old value.
	CompareAndDelete(ctx context.Context, key string, old []byte) error

	// CompareAndSwap atomically updates a key only if its current value matches the expected old value.
	CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error

	// Load retrieves the value for a key. Returns ErrNotExist if the key doesn't exist.
	Load(ctx context.Context, key string) ([]byte, error)

	// LoadAndDelete atomically retrieves and deletes a key's value.
	LoadAndDelete(ctx context.Context, key string) (value []byte, err error)

	// LoadOrStore atomically loads a key's value if it exists, or stores the provided value if it doesn't.
	// Returns the current value and whether it was loaded (true) or stored (false).
	LoadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) (curr []byte, loaded bool, err error)

	// Store sets a key's value with the specified TTL.
	Store(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// StoreOnce stores a key's value only if the key doesn't already exist.
	StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// TTL returns the remaining time to live for a key.
	// Returns -1 if the key exists but has no expiration.
	// Returns -2 if the key does not exist.
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) (int64, error)
}

// Cache provides a Redis-based implementation of the Cacheable interface.
// It wraps a Redis client and provides atomic cache operations.
type Cache struct {
	client           *redis.Client
	metricsCollector MetricsCollector
}

var _ Cacheable = (*Cache)(nil)

// New creates a new Cache instance with the provided Redis client.
func New(client *redis.Client, collectors ...MetricsCollector) *Cache {
	var collector MetricsCollector
	if len(collectors) > 0 && collectors[0] != nil {
		collector = collectors[0]
	} else {
		collector = &AtomicCacheMetrics{}
	}
	return &Cache{
		client:           client,
		metricsCollector: collector,
	}
}

func (c *Cache) Load(ctx context.Context, key string) ([]byte, error) {
	s, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return []byte(s), nil
}

func (c *Cache) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ok, err := c.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrExists
	}

	return nil
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (c *Cache) LoadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) (curr []byte, loaded bool, err error) {
	v, err := c.client.Do(ctx, "SET", key, value, "NX", "GET", "PX", ttl.Milliseconds()).Result()
	// If the previous value does not exist when GET, then it will be nil.
	// But since we successfully set the value, we skip the error.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	// Safe type assertion to prevent panic
	str, ok := v.(string)
	if !ok {
		return nil, false, ErrUnexpectedType
	}

	return []byte(str), true, nil
}

// LoadAndDelete deletes the value for a key, returning the previous value if
// any. The loaded result reports whether the key was present.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (c *Cache) LoadAndDelete(ctx context.Context, key string) (value []byte, err error) {
	v, err := c.client.Do(ctx, "GETDEL", key).Result()
	if err != nil {
		return value, err
	}

	// Safe type assertion to prevent panic
	str, ok := v.(string)
	if !ok {
		return nil, ErrUnexpectedType
	}

	return []byte(str), nil
}

var compareAndDelete = redis.NewScript(`
	-- KEYS[1]: The key
	-- ARGV[1]: The expected value
	local key = KEYS[1]
	local expected = ARGV[1]
	
	local current = redis.call('GET', key)
	if current == false then
		return redis.error_reply("ERR key does not exist")
	end
	
	if current == expected then
		return redis.call('DEL', key)
	end
	
	return redis.error_reply("ERR value mismatch")
`)

// CompareAndDelete deletes the entry for key if its value is equal to old. The
// old value must be of a comparable type.
// If there is no current value for key in the map, CompareAndDelete returns
// false (even if the old value is the nil interface value).
func (c *Cache) CompareAndDelete(ctx context.Context, key string, old []byte) error {
	keys := []string{key}
	argv := []any{old}
	err := compareAndDelete.Run(ctx, c.client, keys, argv...).Err()

	if err != nil {
		// Check for our custom error messages and convert to appropriate errors
		errStr := err.Error()
		if errStr == "ERR key does not exist" {
			return ErrNotExist
		}
		if errStr == "ERR value mismatch" {
			return ErrValueMismatch
		}
		return err
	}

	return nil
}

var compareAndSwap = redis.NewScript(`
	-- KEYS[1]: The key
	-- ARGV[1]: The expected old value
	-- ARGV[2]: The new value
	-- ARGV[3]: The TTL in milliseconds
	local key = KEYS[1]
	local expected = ARGV[1]
	local new_value = ARGV[2]
	local ttl = tonumber(ARGV[3])
	
	local current = redis.call('GET', key)
	if current == false then
		return redis.error_reply("ERR key does not exist")
	end
	
	if current == expected then
		if ttl > 0 then
			return redis.call('SET', key, new_value, 'PX', ttl)
		else
			return redis.call('SET', key, new_value)
		end
	end
	
	return redis.error_reply("ERR value mismatch")
`)

// CompareAndSwap swaps the old and new values for key if the value stored in
// the map is equal to old. The old value must be of a comparable type.
func (c *Cache) CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
	keys := []string{key}
	argv := []any{old, value, ttl.Milliseconds()}
	err := compareAndSwap.Run(ctx, c.client, keys, argv...).Err()

	if err != nil {
		// Check for our custom error messages and convert to appropriate errors
		errStr := err.Error()
		if errStr == "ERR key does not exist" {
			return ErrNotExist
		}
		if errStr == "ERR value mismatch" {
			return ErrValueMismatch
		}
		return err
	}

	return nil
}

// Exists checks if a key exists in the cache.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// TTL returns the remaining time to live for a key.
// Returns -1 if the key exists but has no expiration.
// Returns -2 if the key does not exist.
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// Delete removes one or more keys from the cache.
func (c *Cache) Delete(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Del(ctx, keys...).Result()
}

// Example Prometheus integration
//
// import (
//   "github.com/prometheus/client_golang/prometheus"
//   "github.com/prometheus/client_golang/prometheus/promhttp"
//   "github.com/redis/go-redis/v9"
//   "github.com/alextanhongpin/core/dsync/cache"
//   "net/http"
// )
//
// func main() {
//   totalRequests := prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_total_requests", Help: "Total cache requests."})
//   hits := prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_hits", Help: "Cache hits."})
//   misses := prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_misses", Help: "Cache misses."})
//   sets := prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_sets", Help: "Cache sets."})
//   deletes := prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_deletes", Help: "Cache deletes."})
//   prometheus.MustRegister(totalRequests, hits, misses, sets, deletes)
//
//   metrics := &cache.PrometheusCacheMetrics{
//     TotalRequests: totalRequests,
//     Hits: hits,
//     Misses: misses,
//     Sets: sets,
//     Deletes: deletes,
//   }
//   rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//   c := cache.New(rdb, metrics)
//   http.Handle("/metrics", promhttp.Handler())
//   http.ListenAndServe(":8080", nil)
// }
