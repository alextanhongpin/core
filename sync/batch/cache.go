package batch

import (
	"context"
	"sync"
	"time"
)

type cache[K comparable, V any] interface {
	LoadMany(ctx context.Context, keys ...K) (map[K]V, error)
	StoreMany(ctx context.Context, kv map[K]V, ttl time.Duration) error
}

var _ cache[int, any] = (*Cache[int, any])(nil)

type Cache[K comparable, V any] struct {
	mu   sync.Mutex
	data map[K]*value[V]
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		data: make(map[K]*value[V]),
	}
}

func (c *Cache[K, V]) StoreMany(ctx context.Context, kv map[K]V, ttl time.Duration) error {
	c.mu.Lock()
	for k, v := range kv {
		c.data[k] = newValue(v, ttl)
	}
	c.mu.Unlock()

	return nil
}

func (c *Cache[K, V]) LoadMany(ctx context.Context, ks ...K) (map[K]V, error) {
	m := make(map[K]V)

	c.mu.Lock()
	for _, k := range ks {
		v, ok := c.data[k]
		if !ok {
			continue
		}
		if v.expired() {
			delete(c.data, k)
			continue
		}

		m[k] = v.data
	}
	c.mu.Unlock()

	return m, nil
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
