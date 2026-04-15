package cache

import (
	"context"
	"time"
)

// Storage defines the interface for cache operations with atomic guarantees.
// All operations are thread-safe and provide strong consistency through cache.
type Storage[T any] interface {
	Close() error

	// CompareAndDelete atomically deletes a key only if its current value matches the expected old value.
	CompareAndDelete(ctx context.Context, key string, old T) error

	// CompareAndSwap atomically updates a key only if its current value matches the expected old value.
	CompareAndSwap(ctx context.Context, key string, old, value T, ttl time.Duration) error

	// Load retrieves the value for a key. Returns ErrNotExist if the key doesn't exist.
	Load(ctx context.Context, key string) (T, error)

	// LoadAndDelete atomically retrieves and deletes a key's value.
	LoadAndDelete(ctx context.Context, key string) (value T, err error)

	// LoadOrStore atomically loads a key's value if it exists, or stores the provided value if it doesn't.
	// Returns the current value and whether it was loaded (true) or stored (false).
	LoadOrStore(ctx context.Context, key string, value T, ttl time.Duration) (curr T, loaded bool, err error)

	LoadOrStoreFunc(ctx context.Context, key string, fn func(ctx context.Context, key string) (T, time.Duration, error)) (curr T, loaded bool, err error)

	// Store sets a key's value with the specified TTL.
	Store(ctx context.Context, key string, value T, ttl time.Duration) error

	// StoreOnce stores a key's value only if the key doesn't already exist.
	StoreOnce(ctx context.Context, key string, value T, ttl time.Duration) error

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
