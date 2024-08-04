package cache

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var ErrNotExist = errors.New("cache: not exist")

type Cacheable struct {
	client *redis.Client
}

func New(client *redis.Client) *Cacheable {
	return &Cacheable{
		client: client,
	}
}

func (c *Cacheable) Load(ctx context.Context, key string) (value string, err error) {
	s, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotExist
	}

	return s, err
}

func (c *Cacheable) Store(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (c *Cacheable) LoadOrStore(ctx context.Context, key, value string, ttl time.Duration) (old string, loaded bool, err error) {
	v, err := c.client.Do(ctx, "SET", key, value, "NX", "GET", "PX", ttl.Milliseconds()).Result()
	// If the previous value does not exist when GET, then it will be nil.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}

	if err != nil {
		return "", false, err
	}

	return v.(string), true, nil
}

// LoadAndDelete deletes the value for a key, returning the previous value if
// any. The loaded result reports whether the key was present.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (c *Cacheable) LoadAndDelete(ctx context.Context, key string) (value string, loaded bool, err error) {
	v, err := c.client.Do(ctx, "GETDEL", key).Result()
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}

	return v.(string), true, nil
}

var compareAndDelete = redis.NewScript(`
	-- KEYS[1]: The key
	-- ARGV[1]: The value
	local key = KEYS[1]
	local val = ARGV[1]

	if redis.call('GET', key) == val then
		return redis.call('DEL', key)
	end

	return nil
`)

// CompareAndDelete deletes the entry for key if its value is equal to old. The
// old value must be of a comparable type.
// If there is no current value for key in the map, CompareAndDelete returns
// false (even if the old value is the nil interface value).
func (c *Cacheable) CompareAndDelete(ctx context.Context, key, old string) (deleted bool, err error) {
	keys := []string{key}
	argv := []any{old}
	err = compareAndDelete.Run(ctx, c.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

var compareAndSwap = redis.NewScript(`
	-- KEYS[1]: The key
	-- ARGV[1]: The old value
	-- ARGV[2]: The new value
	-- ARGV[3]: The period in milliseconds.
	local key = KEYS[1]
	local old = ARGV[1]
	local new = ARGV[2]
	local ttl = ARGV[3]

	if redis.call('GET', key) == old then
		return redis.call('SET', key, new, 'PX', ttl)
	end

	return nil
`)

// CompareAndSwap swaps the old and new values for key if the value stored in
// the map is equal to old. The old value must be of a comparable type.
func (c *Cacheable) CompareAndSwap(ctx context.Context, key, old, value string, ttl time.Duration) (swapped bool, err error) {
	keys := []string{key}
	argv := []any{old, value, ttl.Milliseconds()}
	err = compareAndSwap.Run(ctx, c.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
