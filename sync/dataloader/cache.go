package dataloader

import (
	"errors"
	"sync"
)

var ErrNotExist = errors.New("dataloader: key does not exist")

type cache[K comparable, V any] interface {
	// TTL should be up to client implementation.
	Set(key K, value V, err error)
	Get(key K) (V, error)
}

type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	cache map[K]*result[V]
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		cache: make(map[K]*result[V]),
	}
}

func (c *Cache[K, V]) Set(key K, value V, err error) {
	c.mu.Lock()
	c.cache[key] = &result[V]{val: value, err: err}
	c.mu.Unlock()
}

func (c *Cache[K, V]) Get(key K) (V, error) {
	c.mu.RLock()
	r, ok := c.cache[key]
	c.mu.RUnlock()

	if ok {
		return r.unwrap()
	}

	var v V
	return v, ErrNotExist
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	c.cache = make(map[K]*result[V])
	c.mu.Unlock()
}

// Size returns the number of entries in the cache.
func (c *Cache[K, V]) Size() int {
	c.mu.RLock()
	size := len(c.cache)
	c.mu.RUnlock()
	return size
}

// Delete removes a specific key from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

type result[T any] struct {
	val T
	err error
}

func (r *result[T]) unwrap() (T, error) {
	return r.val, r.err
}
