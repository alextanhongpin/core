package cache

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"
)

// Separator builds and parses composite keys such as "users:42".
//
//	sep := cache.Separator(':')
//	key := sep.Join("users", "42") // "users:42"
//	parts := sep.Split(key)        // ["users" "42"]
type Separator rune

// Join joins s with the separator.
func (sep Separator) Join(s ...string) string {
	return strings.Join(s, string(sep))
}

// Split splits s on the separator.
func (sep Separator) Split(s string) []string {
	return strings.Split(s, string(sep))
}

var _ cache[any] = (*Cache[any])(nil)

// Config configures a [Cache].
type Config struct {
	// Storage is the byte backend, for example [NewRedis], [NewFile] or
	// [NewFS]. Required. The Cache does not copy it: [Cache.Close] calls
	// Storage.Close, so close only one of them when several caches share a
	// backend.
	Storage Storage[[]byte]

	// Codec encodes values to bytes. Optional; defaults to [JSONCodec]. It
	// must be safe for concurrent use.
	Codec Codec
}

// WithDefaults returns a copy of c with unset optional fields filled in.
func (c Config) WithDefaults() Config {
	if c.Codec == nil {
		c.Codec = NewJSONCodec()
	}
	return c
}

// Validate reports whether c has all required fields.
func (c Config) Validate() error {
	if c.Storage == nil {
		return errors.New("cache: storage is required")
	}
	if c.Codec == nil {
		return errors.New("cache: codec is required")
	}
	return nil
}

// Cache stores values of type T in a byte [Storage], encoding them with a
// [Codec]. It is safe for concurrent use. Its methods behave like the
// corresponding [Storage] methods; see there for TTL and error conventions.
type Cache[T any] struct {
	storage C[[]byte]
	codec   Codec
}

// New returns a Cache for values of type T. It returns an error if
// cfg.Storage is nil.
func New[T any](cfg Config) (*Cache[T], error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Cache[T]{storage: cfg.Storage, codec: cfg.Codec}, nil
}

// MustNew is like [New] but panics on error. Use it during program startup.
func MustNew[T any](cfg Config) *Cache[T] {
	c, err := New[T](cfg)
	if err != nil {
		panic(err)
	}
	return c
}

// Close closes the underlying Storage. Closing a [Redis] storage does not
// close the Redis client.
func (c *Cache[T]) Close() error {
	return c.storage.Close()
}

// CompareAndDelete deletes key only if its current value equals old. Values
// are compared by their encoded bytes, so the codec must encode equal values
// identically (JSON does for structs). It returns [ErrNotExist] if the key is
// missing or the value differs.
func (c *Cache[T]) CompareAndDelete(ctx context.Context, key string, old T) error {
	b, err := c.marshal(old)
	if err != nil {
		return err
	}

	return c.storage.CompareAndDelete(ctx, key, b)
}

// CompareAndSwap replaces key's value only if the current value equals old,
// compared by encoded bytes. It returns [ErrNotExist] if the key is missing or
// the value differs.
func (c *Cache[T]) CompareAndSwap(ctx context.Context, key string, old, value T, ttl time.Duration) error {
	a, err := c.marshal(old)
	if err != nil {
		return err
	}
	b, err := c.marshal(value)
	if err != nil {
		return err
	}

	return c.storage.CompareAndSwap(ctx, key, a, b, ttl)
}

// Delete removes keys and returns how many existed.
func (c *Cache[T]) Delete(ctx context.Context, keys ...string) (int64, error) {
	return c.storage.Delete(ctx, keys...)
}

// Exists reports whether key exists.
func (c *Cache[T]) Exists(ctx context.Context, key string) (bool, error) {
	return c.storage.Exists(ctx, key)
}

// Expire sets a new TTL on an existing key; ttl <= 0 deletes it. It returns
// [ErrNotExist] if the key does not exist.
func (c *Cache[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.storage.Expire(ctx, key, ttl)
}

// Load returns the decoded value for key, or [ErrNotExist].
func (c *Cache[T]) Load(ctx context.Context, key string) (T, error) {
	var zero T
	b, err := c.storage.Load(ctx, key)
	if err != nil {
		return zero, err
	}

	t, err := c.unmarshal(b)
	if err != nil {
		return zero, err
	}
	return t, nil
}

// LoadAndDelete atomically returns and deletes the value for key, or returns
// [ErrNotExist].
func (c *Cache[T]) LoadAndDelete(ctx context.Context, key string) (T, error) {
	var zero T
	b, err := c.storage.LoadAndDelete(ctx, key)
	if err != nil {
		return zero, err
	}

	t, err := c.unmarshal(b)
	if err != nil {
		return zero, err
	}
	return t, nil
}

// LoadOrStore returns the existing value for key (loaded=true), or stores
// value and returns it (loaded=false).
func (c *Cache[T]) LoadOrStore(ctx context.Context, key string, value T, ttl time.Duration) (curr T, loaded bool, err error) {
	var zero T
	b, err := c.marshal(value)
	if err != nil {
		return zero, false, err
	}

	b, loaded, err = c.storage.LoadOrStore(ctx, key, b, ttl)
	if err != nil {
		return zero, false, err
	}

	t, err := c.unmarshal(b)
	if err != nil {
		return zero, false, err
	}

	return t, loaded, nil
}

// Store encodes value and stores it under key. ttl <= 0 means no expiry.
func (c *Cache[T]) Store(ctx context.Context, key string, value T, ttl time.Duration) error {
	b, err := c.marshal(value)
	if err != nil {
		return err
	}

	return c.storage.Store(ctx, key, b, ttl)
}

// StoreOnce stores value only if key does not exist; otherwise it returns
// [ErrExists].
func (c *Cache[T]) StoreOnce(ctx context.Context, key string, value T, ttl time.Duration) error {
	b, err := c.marshal(value)
	if err != nil {
		return err
	}

	return c.storage.StoreOnce(ctx, key, b, ttl)
}

// TTL returns the remaining time to live for a key: -1 if the key has no
// expiry, -2 if it does not exist.
func (c *Cache[T]) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.storage.TTL(ctx, key)
}

// LoadOrCreate returns the cached value for key (loaded=true). On a miss it
// calls create, stores the result with the TTL create returns, and returns it
// (loaded=false). Concurrent callers in the same process share one create
// call. To run create once across processes, use the Cache type in
// github.com/alextanhongpin/core/dsync/singleflight.
func (c *Cache[T]) LoadOrCreate(ctx context.Context, key string, create func(context.Context, string) (T, time.Duration, error)) (value T, loaded bool, err error) {
	b, loaded, err := c.storage.LoadOrCreate(ctx, key, func(ctx context.Context, key string) ([]byte, time.Duration, error) {
		v, ttl, err := create(ctx, key)
		if err != nil {
			return nil, 0, err
		}
		b, err := c.marshal(v)
		if err != nil {
			return nil, 0, err
		}
		return b, ttl, nil
	})
	var zero T
	if err != nil {
		return zero, false, err
	}

	// Unmarshal the current value (either from storage or what we just stored)
	value, err = c.unmarshal(b)
	if err != nil {
		return zero, false, err
	}

	return value, loaded, nil
}

func (c *Cache[T]) marshal(v T) ([]byte, error) {
	var b bytes.Buffer
	err := c.codec.NewEncoder(&b).Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c *Cache[T]) unmarshal(b []byte) (T, error) {
	var v T
	err := c.codec.NewDecoder(bytes.NewReader(b)).Decode(&v)
	return v, err
}
