package cache

import (
	"context"
	"io"
	"time"
)

// Storage is the set of operations every backend in this package implements
// ([Redis], [File] and [FS] for raw bytes, [Cache] for typed values).
//
// TTL conventions are the same for all backends:
//   - ttl > 0 expires the key after ttl.
//   - ttl <= 0 stores the key without expiry (it lives until deleted).
//
// "Compare" operations (CompareAndSwap, CompareAndDelete) return [ErrNotExist]
// both when the key is missing and when its value does not match old.
//
// Atomicity and persistence guarantees depend on the backend. Implementations
// are safe for concurrent use.
type Storage[T any] interface {
	io.Closer

	// CompareAndDelete deletes key only if its current value equals old.
	// It returns ErrNotExist if the key is missing or the value differs.
	CompareAndDelete(ctx context.Context, key string, old T) error

	// CompareAndSwap replaces key's value with value only if the current value
	// equals old. It returns ErrNotExist if the key is missing or the value
	// differs.
	CompareAndSwap(ctx context.Context, key string, old, value T, ttl time.Duration) error

	// Load returns the value for key, or ErrNotExist.
	Load(ctx context.Context, key string) (T, error)

	// LoadAndDelete atomically returns and deletes key's value, or returns
	// ErrNotExist.
	LoadAndDelete(ctx context.Context, key string) (value T, err error)

	// LoadOrStore returns the existing value (loaded=true) or stores value and
	// returns it (loaded=false).
	LoadOrStore(ctx context.Context, key string, value T, ttl time.Duration) (curr T, loaded bool, err error)

	// LoadOrCreate returns the existing value (loaded=true). On a miss it calls
	// fn once per key within this process, stores the result with the TTL fn
	// returns, and returns it (loaded=false). Concurrent callers for the same
	// key wait for the first caller and share its result and error. Use
	// [Lock.LoadOrCreate] to coordinate fills across processes.
	LoadOrCreate(ctx context.Context, key string, fn func(ctx context.Context, key string) (T, time.Duration, error)) (curr T, loaded bool, err error)

	// Store sets key's value, overwriting any existing value.
	Store(ctx context.Context, key string, value T, ttl time.Duration) error

	// StoreOnce sets key's value only if the key does not exist; otherwise it
	// returns ErrExists.
	StoreOnce(ctx context.Context, key string, value T, ttl time.Duration) error

	// Exists reports whether key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// TTL returns the remaining time to live for key. Like Redis PTTL it
	// returns -1 (nanosecond) if the key has no expiry and -2 if the key does
	// not exist.
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Expire sets a new TTL on an existing key; ttl <= 0 deletes the key.
	// It returns ErrNotExist if the key does not exist.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Delete removes keys and returns how many existed. Missing keys are not
	// an error.
	Delete(ctx context.Context, keys ...string) (int64, error)
}

// C is a short alias for [Storage], kept for backward compatibility.
type C[T any] = Storage[T]

type cache[T any] = Storage[T]
