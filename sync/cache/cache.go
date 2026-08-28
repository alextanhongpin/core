package cache

import (
	"runtime"
	"sync"
	"weak"
)

// https://go.dev/blog/cleanups-and-weak
type Cache[K comparable, V any] struct {
	create func(K) (*V, error)
	m      sync.Map
}

func New[K comparable, V any](create func(K) (*V, error)) *Cache[K, V] {
	return &Cache[K, V]{create: create}
}

func (c *Cache[K, V]) Get(key K) (*V, error) {
	var newValue *V
	for {
		// Try to load an existing value out of the cache.
		value, ok := c.m.Load(key)
		if !ok {
			// No value found. Create a new mapped file if needed.
			if newValue == nil {
				var err error
				newValue, err = c.create(key)
				if err != nil {
					return nil, err
				}
			}

			// Try to install the new mapped file.
			wp := weak.Make(newValue)
			var loaded bool
			value, loaded = c.m.LoadOrStore(key, wp)
			if !loaded {
				runtime.AddCleanup(newValue, func(key K) {
					// Only delete if the weak pointer is equal. If it's not, someone
					// else already deleted the entry and installed a new mapped file.
					c.m.CompareAndDelete(key, wp)
				}, key)
				return newValue, nil
			}
		}

		// See if our cache entry is valid.
		if mf := value.(weak.Pointer[V]).Value(); mf != nil {
			return mf, nil
		}

		// Discovered a nil entry awaiting cleanup. Eagerly delete it.
		c.m.CompareAndDelete(key, value)
	}
}
