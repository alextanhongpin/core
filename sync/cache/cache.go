package cache

import (
	"errors"
	"runtime"
	"sync"
	"weak"
)

// Cache holds weak references to values, keyed by K. The cache never keeps a
// value alive: once no caller holds a strong reference, the garbage collector
// may reclaim it and a later LoadOrCreate for the same key calls create again.
// Callers must retain returned values for as long as they need them (or need
// the cache to return the same instance).
//
// Concurrent misses for the same key may each invoke create; only one result is
// installed and the others are discarded, so create must be safe to run more
// than once per key and should be cheap or idempotent. Cache is safe for
// concurrent use. It is used by sync/dataloader, sync/promise and dsync, so
// its API is kept stable.
// https://go.dev/blog/cleanups-and-weak
type Cache[K comparable, V any] struct {
	create func(K) (*V, error)
	m      sync.Map
}

// New returns an empty Cache that calls create to build values on a miss.
func New[K comparable, V any](create func(K) (*V, error)) *Cache[K, V] {
	return &Cache[K, V]{create: create}
}

// LoadOrCreate returns the live value for key, calling create on a miss. The
// bool reports whether the value was already cached (true) or is the one this
// call just created and installed (false). If create returns an error, or a nil
// value, the error is returned and nothing is stored. When several callers race
// on a miss, create may run in each of them; callers that lose the race receive
// the winner's value with loaded set to true and their own value is dropped.
func (c *Cache[K, V]) LoadOrCreate(key K) (*V, bool, error) {
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
					return nil, false, err
				}
				if newValue == nil {
					return nil, false, errors.New("cache: create returned nil value")
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
				return newValue, false, nil
			}
		}

		// See if our cache entry is valid.
		if mf := value.(weak.Pointer[V]).Value(); mf != nil {
			return mf, true, nil
		}

		// Discovered a nil entry awaiting cleanup. Eagerly delete it.
		c.m.CompareAndDelete(key, value)
	}
}
