// Package cache provides JSON serialization wrapper for the cache interface.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"

	"golang.org/x/sync/singleflight"
)

// JSON-specific errors
var (
	ErrOperationNotSupported = errors.New("operation not supported by underlying cache implementation")
)

// JSON provides automatic JSON marshaling/unmarshaling for cache operations.
// It wraps a Cacheable implementation and handles serialization transparently.
// All methods that take value parameters expect pointers for unmarshaling operations.
type JSON struct {
	Cache Cacheable
	Group *singleflight.Group
}

// NewJSON creates a new JSON cache wrapper with the provided Redis client.
// The returned JSON cache provides automatic serialization/deserialization
// for Go structs and values.
func NewJSON(client *redis.Client) *JSON {
	return &JSON{
		Cache: New(client),
		Group: new(singleflight.Group),
	}
}

// Load retrieves and unmarshals a JSON value from the cache.
// The value parameter should be a pointer to the destination type.
func (s *JSON) Load(ctx context.Context, key string, v any) error {
	b, err := s.Cache.Load(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &v)
}

// Store marshals and stores a JSON value in the cache with the specified TTL.
func (s *JSON) Store(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Cache.Store(ctx, key, b, ttl)
}

// StoreOnce stores a key's JSON value only if the key doesn't already exist.
func (s *JSON) StoreOnce(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Cache.StoreOnce(ctx, key, b, ttl)
}

// LoadAndDelete atomically retrieves and deletes a JSON value from the cache.
// The value parameter should be a pointer to the destination type.
func (s *JSON) LoadAndDelete(ctx context.Context, key string, value any) error {
	b, err := s.Cache.LoadAndDelete(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &value)
}

// CompareAndDelete atomically deletes a key only if its current JSON value matches the expected old value.
func (s *JSON) CompareAndDelete(ctx context.Context, key string, old any) error {
	b, err := json.Marshal(old)
	if err != nil {
		return err
	}

	return s.Cache.CompareAndDelete(ctx, key, b)
}

// CompareAndSwap atomically updates a key only if its current JSON value matches the expected old value.
func (s *JSON) CompareAndSwap(ctx context.Context, key string, old, value any, ttl time.Duration) error {
	a, err := json.Marshal(old)
	if err != nil {
		return err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Cache.CompareAndSwap(ctx, key, a, b, ttl)
}

type PrefixKey struct {
	Prefix string
	Key    string
}

func (k PrefixKey) String() string {
	return fmt.Sprintf("%s:%s", k.Prefix, k.Key)
}

type Item struct {
	Key   string
	TTL   time.Duration
	Value any
}

func (s *JSON) LoadOrStore(ctx context.Context, key fmt.Stringer, value any, getter func(ctx context.Context, key fmt.Stringer) (*Item, error)) (loaded bool, err error) {
	// First, try to load from cache
	err = s.Load(ctx, key.String(), value)
	if err == nil {
		return true, nil
	}
	// If the error is not ErrNotExist, return the error
	if !errors.Is(err, ErrNotExist) {
		return false, err
	}

	res, err, shared := s.Group.Do(key.String(), func() (any, error) {
		// Call getter to get the value to store
		i, err := getter(ctx, key)
		if err != nil {
			return false, err
		}

		// Marshal the value from getter
		b, err := json.Marshal(i.Value)
		if err != nil {
			return false, err
		}

		// Use atomic LoadOrStore operation to prevent race conditions
		curr, loaded, err := s.Cache.LoadOrStore(ctx, i.Key, b, i.TTL)
		if err != nil {
			return false, err
		}

		// Unmarshal the current value (either from cache or what we just stored)
		if err := json.Unmarshal(curr, value); err != nil {
			return false, err
		}

		return loaded, nil
	})
	if err != nil {
		return false, err
	}

	return res.(bool) && shared, nil
}

// Exists checks if a key exists in the cache.
func (s *JSON) Exists(ctx context.Context, key string) (bool, error) {
	if c, ok := s.Cache.(*Cache); ok {
		return c.Exists(ctx, key)
	}
	// Fallback for other Cacheable implementations
	_, err := s.Cache.Load(ctx, key)
	if errors.Is(err, ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// TTL returns the remaining time to live for a key.
func (s *JSON) TTL(ctx context.Context, key string) (time.Duration, error) {
	if c, ok := s.Cache.(*Cache); ok {
		return c.TTL(ctx, key)
	}
	return 0, ErrOperationNotSupported
}

// Delete removes one or more keys from the cache.
func (s *JSON) Delete(ctx context.Context, keys ...string) (int64, error) {
	if c, ok := s.Cache.(*Cache); ok {
		return c.Delete(ctx, keys...)
	}
	return 0, ErrOperationNotSupported
}

// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
func (s *JSON) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if c, ok := s.Cache.(*Cache); ok {
		return c.Expire(ctx, key, ttl)
	}
	return ErrOperationNotSupported
}
