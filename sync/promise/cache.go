package promise

import (
	"runtime"
	"sync"
	"weak"
)

// https://go.dev/blog/cleanups-and-weak
type Cache[K comparable, V any] struct {
	create func(K) (*V, error)
	sync.Map
}

func NewCache[K comparable, V any](create func(K) (*V, error)) *Cache[K, V] {
	return &Cache[K, V]{create: create}
}

func (c *Cache[K, V]) LoadOrCreate(key K) (v *V, loaded bool, err error) {
	var newValue *V
	for {
		// Try to load an existing value out of the cache.
		value, ok := c.Load(key)
		if !ok {
			// No value found. Create a new mapped file if needed.
			if newValue == nil {
				var err error
				newValue, err = c.create(key)
				if err != nil {
					return nil, false, err
				}
			}

			// Try to install the new mapped file.
			wp := weak.Make(newValue)
			var loaded bool
			value, loaded = c.LoadOrStore(key, wp)
			if !loaded {
				runtime.AddCleanup(newValue, func(key K) {
					// Only delete if the weak pointer is equal. If it's not, someone
					// else already deleted the entry and installed a new mapped file.
					c.CompareAndDelete(key, wp)
				}, key)
				return newValue, false, nil
			}
		}

		// See if our cache entry is valid.
		if mf := value.(weak.Pointer[V]).Value(); mf != nil {
			return mf, true, nil
		}

		// Discovered a nil entry awaiting cleanup. Eagerly delete it.
		c.CompareAndDelete(key, value)
	}
}

func (c *Cache[K, V]) LoadAndDelete(key K) (*V, bool) {
	value, loaded := c.Map.LoadAndDelete(key)
	if !loaded {
		return nil, false
	}

	// See if our cache entry is valid.
	if v := value.(weak.Pointer[V]).Value(); v != nil {
		return v, true
	}

	return nil, false
}
