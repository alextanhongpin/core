// Package storage provides Cache serialization wrapper for the storage interface.
package cache

import (
	"context"
	"strings"
	"time"
)

type Separator rune

func (sep Separator) Join(s ...string) string {
	return strings.Join(s, string(sep))
}

func (sep Separator) Split(s string) []string {
	return strings.Split(s, string(sep))
}

type Encoder interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

var _ Storage[any] = (*Cache[any])(nil)

// Cache provides automatic Cache marshaling/unmarshaling for storage operations.
// It wraps a Storage implementation and handles serialization transparently.
// All methods that take value parameters expect pointers for unmarshaling operations.
type Cache[T any] struct {
	Storage Storage[[]byte]
	Encoder Encoder
}

// New creates a new Cache storage wrapper with the given Redis client.
// The returned Cache storage provides automatic serialization/deserialization
// for Go structs and values.
func New[T any]() *Cache[T] {
	return &Cache[T]{
		Encoder: NewJSONEncoder(),
	}
}

func (c *Cache[T]) Close() error {
	return c.Storage.Close()
}

// CompareAndDelete atomically deletes a key only if its current Cache value matches the expected old value.
func (c *Cache[T]) CompareAndDelete(ctx context.Context, key string, old T) error {
	b, err := c.Encoder.Marshal(old)
	if err != nil {
		return err
	}

	return c.Storage.CompareAndDelete(ctx, key, b)
}

// CompareAndSwap atomically updates a key only if its current Cache value matches the expected old value.
func (c *Cache[T]) CompareAndSwap(ctx context.Context, key string, old, value T, ttl time.Duration) error {
	a, err := c.Encoder.Marshal(old)
	if err != nil {
		return err
	}
	b, err := c.Encoder.Marshal(value)
	if err != nil {
		return err
	}

	return c.Storage.CompareAndSwap(ctx, key, a, b, ttl)
}

// Delete removes one or more keys from the storage.
func (c *Cache[T]) Delete(ctx context.Context, keys ...string) (int64, error) {
	return c.Storage.Delete(ctx, keys...)
}

// Exists checks if a key exists in the storage.
func (c *Cache[T]) Exists(ctx context.Context, key string) (bool, error) {
	return c.Storage.Exists(ctx, key)
}

// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
func (c *Cache[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.Storage.Expire(ctx, key, ttl)
}

// Load retrieves and unmarshals a Cache value from the storage.
// The value parameter should be a pointer to the destination type.
func (c *Cache[T]) Load(ctx context.Context, key string) (T, error) {
	var zero T
	b, err := c.Storage.Load(ctx, key)
	if err != nil {
		return zero, err
	}

	var t T
	err = c.Encoder.Unmarshal(b, &t)
	if err != nil {
		return zero, err
	}
	return t, nil
}

// LoadAndDelete atomically retrieves and deletes a Cache value from the storage.
// The value parameter should be a pointer to the destination type.
func (c *Cache[T]) LoadAndDelete(ctx context.Context, key string) (T, error) {
	var zero T
	b, err := c.Storage.LoadAndDelete(ctx, key)
	if err != nil {
		return zero, err
	}
	var t T
	err = c.Encoder.Unmarshal(b, &t)
	if err != nil {
		return zero, err
	}
	return t, nil
}

// LoadOrStore atomically loads a key's value if it exists, or stores the given value if it doesn't.
func (c *Cache[T]) LoadOrStore(ctx context.Context, key string, value T, ttl time.Duration) (curr T, loaded bool, err error) {
	var zero T
	b, err := c.Encoder.Marshal(value)
	if err != nil {
		return zero, false, err
	}

	b, loaded, err = c.Storage.LoadOrStore(ctx, key, b, ttl)
	if err != nil {
		return zero, false, err
	}

	var t T
	err = c.Encoder.Unmarshal(b, &t)
	if err != nil {
		return zero, false, err
	}

	return t, loaded, nil
}

// Store marshals and stores a Cache value in the storage with the specified TTL.
func (c *Cache[T]) Store(ctx context.Context, key string, value T, ttl time.Duration) error {
	b, err := c.Encoder.Marshal(value)
	if err != nil {
		return err
	}

	return c.Storage.Store(ctx, key, b, ttl)
}

// StoreOnce stores a key'c Cache value only if the key doesn't already exist.
func (c *Cache[T]) StoreOnce(ctx context.Context, key string, value T, ttl time.Duration) error {
	b, err := c.Encoder.Marshal(value)
	if err != nil {
		return err
	}

	return c.Storage.StoreOnce(ctx, key, b, ttl)
}

// TTL returns the remaining time to live for a key.
// Returns -1 if the key exists but has no expiration.
// Returns -2 if the key does not exist.
func (c *Cache[T]) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.Storage.TTL(ctx, key)
}

func (c *Cache[T]) LoadOrStoreFunc(ctx context.Context, key string, getter func(context.Context, string) (T, time.Duration, error)) (value T, loaded bool, err error) {
	b, loaded, err := c.Storage.LoadOrStoreFunc(ctx, key, func(ctx context.Context, key string) ([]byte, time.Duration, error) {
		v, ttl, err := getter(ctx, key)
		if err != nil {
			return nil, 0, err
		}
		b, err := c.Encoder.Marshal(v)
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
	err = c.Encoder.Unmarshal(b, &value)
	if err != nil {
		return zero, false, err
	}

	return value, loaded, nil
}
