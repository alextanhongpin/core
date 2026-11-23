package cache

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
	"golang.org/x/sync/singleflight"
)

// Redis is a [Storage] backed by Redis string keys. Every operation is a
// single atomic Redis command, so it is safe to share between processes.
// Values are stored under the exact key you pass; add your own prefix to avoid
// clashes. CompareAndSwap and CompareAndDelete require Redis 8.4+.
type Redis struct {
	client *redis.Client
	group  singleflight.Group
}

var _ cache[[]byte] = (*Redis)(nil)

// NewRedis returns a Redis storage that uses client. The client stays owned
// by the caller: [Redis.Close] does nothing, so close the client yourself.
func NewRedis(client *redis.Client) (*Redis, error) {
	if client == nil {
		return nil, errors.New("cache: Redis client is required")
	}
	return &Redis{client: client}, nil
}

// MustNewRedis is like [NewRedis] but panics on error.
func MustNewRedis(client *redis.Client) *Redis {
	r, err := NewRedis(client)
	if err != nil {
		panic(err)
	}
	return r
}

// Close does nothing; the Redis client is owned by the caller.
func (r *Redis) Close() error { return nil }

// Load returns the value for key, or [ErrNotExist].
func (r *Redis) Load(ctx context.Context, key string) ([]byte, error) {
	s, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotExist
	}
	if err != nil {
		return nil, err
	}

	return []byte(s), nil
}

// Store sets a key's value with the specified TTL.
func (r *Redis) Store(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, noExpiry(ttl)).Err()
}

// StoreOnce stores a key's value only if the key doesn't already exist;
// otherwise it returns [ErrExists].
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

// LoadOrCreate returns the existing value for key (loaded=true). On a miss it
// calls create, stores the result with SET NX and returns it (loaded=false).
// Concurrent callers in this process share one create call, which runs with
// the first caller's context. Other processes may call create at the same
// time; the first SET wins and the others load its value. Use
// github.com/alextanhongpin/core/dsync/singleflight if create must run only
// once across processes.
func (r *Redis) LoadOrCreate(ctx context.Context, key string, create func(context.Context, string) ([]byte, time.Duration, error)) (curr []byte, loaded bool, err error) {
	if create == nil {
		return nil, false, errors.New("cache: create is required")
	}
	val, err := r.Load(ctx, key)
	if err == nil {
		return val, true, nil
	}
	if !errors.Is(err, ErrNotExist) {
		return nil, false, err
	}

	var created atomic.Bool
	// The "shared" value will be true if all the values are shared.
	res, err, _ := r.group.Do(key, func() (any, error) {
		val, ttl, err := create(ctx, key)
		if err != nil {
			return nil, err
		}

		curr, loaded, err = r.LoadOrStore(ctx, key, val, ttl)
		if err != nil {
			return nil, err
		}
		if !loaded {
			created.Store(true)
		}

		return curr, nil
	})
	if err != nil {
		return nil, false, err
	}

	// Callers share the singleflight result; give each its own copy.
	return slices.Clone(res.([]byte)), !created.Load(), nil
}

// LoadAndDelete atomically returns and deletes the value for key (GETDEL), or
// returns [ErrNotExist].
func (r *Redis) LoadAndDelete(ctx context.Context, key string) (value []byte, err error) {
	s, err := r.client.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return value, ErrNotExist
	}
	if err != nil {
		return value, err
	}

	return []byte(s), nil
}

// CompareAndDelete deletes key only if its value equals old (DELEX IFDEQ). It
// returns [ErrNotExist] if the key is missing or the value differs.
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

// CompareAndSwap sets key to value only if its current value equals old (SET
// IFDEQ). It returns [ErrNotExist] if the key is missing or the value differs.
// The new value gets ttl; ttl <= 0 removes any expiry.
func (r *Redis) CompareAndSwap(ctx context.Context, key string, old, value []byte, ttl time.Duration) error {
	_, err := r.client.SetIFDEQ(ctx, key, value, helper.DigestBytes(old), noExpiry(ttl)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrNotExist
	}
	return err
}

// Exists reports whether key exists.
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// TTL returns the remaining time to live for key with millisecond precision
// (PTTL): -1 if the key has no expiry, -2 if it does not exist.
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.PTTL(ctx, key).Result()
}

// Expire sets a new TTL on an existing key (PEXPIRE); ttl <= 0 deletes it. It
// returns [ErrNotExist] if the key does not exist.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	ok, err := r.client.PExpire(ctx, key, ttl).Result()
	if err == nil && !ok {
		return ErrNotExist
	}
	return err
}

// Delete removes keys and returns how many existed.
func (r *Redis) Delete(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil // DEL without keys is a Redis syntax error.
	}
	n, err := r.client.Del(ctx, keys...).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrNotExist
	}
	return n, err
}

// noExpiry maps every non-positive TTL to 0 (no expiry). go-redis treats -1ns
// as KEEPTTL, which would silently keep an old expiry instead of clearing it.
func noExpiry(ttl time.Duration) time.Duration {
	if ttl < 0 {
		return 0
	}
	return ttl
}
