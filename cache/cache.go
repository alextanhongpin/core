package cache

// Cache defines the interface for cache implementations.
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte) error
	Delete(key string) error
}

// Func represents a function that computes a value from a key.
type Func func(key string) ([]byte, error)

// WithCache returns a new Func that caches results using the provided Cache.
func (f Func) WithCache(c Cache) Func {
	return func(key string) ([]byte, error) {
		if data, ok := c.Get(key); ok {
			return data, nil
		}
		data, err := f(key)
		if err != nil {
			return nil, err
		}
		if err := c.Set(key, data); err != nil {
			return nil, err
		}
		return data, nil
	}
}
