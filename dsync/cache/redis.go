// Package cache provides a Redis-based cache implementation with atomic operations.
package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
	"golang.org/x/sync/singleflight"
)

// Redis provides a cache-based implementation of the Storage interface.
// It wraps a Redis client and provides atomic cache operations.
type Redis struct {
	client *redis.Client
	group  singleflight.Group
}

var _ Storage[[]byte] = (*Redis)(nil)

// NewRedis creates a new Redis instance with the provided Redis client.
func NewRedis(client *redis.Client) *Redis {
	return &Redis{
		client: client,
	}
}

// Close closes the redis connection.
func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Load(ctx context.Context, key string) ([]byte, error) {
	s, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return []byte(s), nil
}

// Store sets a key's value with the specified TTL.
func (r *Redis) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// StoreOnce stores a key's value only if the key doesn't already
// exist.
func (r *Redis) StoreOnce(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	_, err := r.client.SetArgs(ctx, key, value, redis.SetArgs{
		TTL:  ttl,
		Mode: string(redis.NX),
	}).Result()
	if errors.Is(err, redis.Nil) {
		return ErrExists
	}

	return err
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (r *Redis) LoadOrStore(ctx context.Context, key string, value []byte, ttl time.Duration) (curr []byte, loaded bool, err error) {
	s, err := r.client.SetArgs(ctx, key, value, redis.SetArgs{
		Get:  true,
		Mode: string(redis.NX),
		TTL:  ttl,
	}).Result()
	// If the previous value does not exist when GET, then it will be nil.
	// But since we successfully set the value, we skip the error.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return []byte(s), true, nil
}

// LoadOrStoreFunc returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (r *Redis) LoadOrStoreFunc(ctx context.Context, key string, getter func(context.Context, string) ([]byte, time.Duration, error)) (curr []byte, loaded bool, err error) {
	val, err := r.Load(ctx, key)
	if err == nil {
		return val, true, nil
	}
	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}
	type cache struct {
		hit bool
		val []byte
	}

	var called atomic.Bool
	// The "shared" value will be true if all the values are shared.
	c, err, shared := r.group.Do(key, func() (any, error) {
		called.Store(true)
		val, ttl, err := getter(ctx, key)
		if err != nil {
			return nil, err
		}

		curr, loaded, err = r.LoadOrStore(ctx, key, val, ttl)
		if err != nil {
			return nil, err
		}

		return &cache{
			val: curr,
			hit: loaded,
		}, nil
	})
	if err != nil {
		return nil, false, err
	}
	res := c.(*cache)

	return res.val, res.hit || (!called.Load() && shared), nil
}

// LoadAndDelete deletes the value for a key, returning the previous value if
// any. The loaded result reports whether the key was present.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (r *Redis) LoadAndDelete(ctx context.Context, key string) (value []byte, err error) {
	s, err := r.client.GetDel(ctx, key).Result()
	if err != nil {
		return value, err
	}

	return []byte(s), nil
}

// CompareAndDelete deletes the entry for key if its value is equal to old. The
// old value must be of a comparable type.
// If there is no current value for key in the map, CompareAndDelete returns
// false (even if the old value is the nil interface value).
func (r *Redis) CompareAndDelete(ctx context.Context, key string, old []byte) error {
	n, err := r.client.DelExArgs(ctx, key, redis.DelExArgs{
		Mode:        "IFDEQ",
		MatchDigest: helper.DigestBytes(old),
	}).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotExist
	}
	return nil
}

// CompareAndSwap swaps the old and new values for key if the value stored in
// the map is equal to old. The old value must be of a comparable type.
func (r *Redis) CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
	_, err := r.client.SetIFDEQ(ctx, key, value, helper.DigestBytes(old), ttl).Result()
	return err
}

// Exists checks if a key exists in the cache.
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// TTL returns the remaining time to live for a key.
// Returns -1 if the key exists but has no expiration.
// Returns -2 if the key does not exist.
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Expire sets a timeout on a key. After the timeout has expired, the key will automatically be deleted.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// Delete removes one or more keys from the cache.
func (r *Redis) Delete(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}
