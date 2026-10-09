package singleflight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	redis "github.com/redis/go-redis/v9"
)

// CacheConfig controls cache-fill coordination. Zero durations and suffix use
// defaults. Redis is borrowed; Cache has no Close method.
type CacheConfig struct {
	LockTTL, WaitTTL, PollInterval time.Duration
	Suffix                         string
}

func (c CacheConfig) WithDefaults() CacheConfig {
	g := (Config{LockTTL: c.LockTTL, WaitTTL: c.WaitTTL, PollInterval: c.PollInterval}).WithDefaults()
	c.LockTTL, c.WaitTTL, c.PollInterval = g.LockTTL, g.WaitTTL, g.PollInterval
	if c.Suffix == "" {
		c.Suffix = "fetch"
	}
	return c
}
func (c CacheConfig) Validate() error {
	return (Config{LockTTL: c.LockTTL, WaitTTL: c.WaitTTL, PollInterval: c.PollInterval}).Validate()
}

type Cache[T any] struct {
	client *redis.Client
	group  *Group
	cfg    CacheConfig
}

func NewCache[T any](client *redis.Client, cfg CacheConfig) (*Cache[T], error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	g, err := New(client, Config{LockTTL: cfg.LockTTL, WaitTTL: cfg.WaitTTL, PollInterval: cfg.PollInterval})
	if err != nil {
		return nil, err
	}
	return &Cache[T]{client: client, group: g, cfg: cfg}, nil
}
func MustNewCache[T any](client *redis.Client, cfg CacheConfig) *Cache[T] {
	c, err := NewCache[T](client, cfg)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Cache[T]) LoadOrStore(ctx context.Context, key string, getter func(ctx context.Context) (T, error), ttl time.Duration) (T, bool, error) {
	var zero T
	if key == "" || getter == nil || ttl < 0 || (ttl > 0 && ttl < time.Millisecond) {
		return zero, false, errors.New("singleflight: key, getter, and valid TTL are required")
	}
	t, err := c.load(ctx, key)
	if err == nil {
		return t, true, nil
	}

	if !errors.Is(err, redis.Nil) {
		return t, false, err
	}

	created := false
	did, err := c.group.Do(ctx, fmt.Sprintf("%s:%s", key, c.cfg.Suffix), func(ctx context.Context) error {
		if _, err := c.load(ctx, key); err == nil {
			return nil
		} else if !errors.Is(err, redis.Nil) {
			return err
		}
		v, err := getter(ctx)
		if err != nil {
			return err
		}

		if err := c.store(ctx, key, v, ttl); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return t, false, err
	}

	t, err = c.load(ctx, key)
	if err != nil {
		return t, false, err
	}

	return t, !did || !created, nil
}

func (c *Cache[T]) load(ctx context.Context, key string) (t T, err error) {
	b, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(b, &t)
	return t, err
}

// Publication and ownership validation share one Redis command. Both keys must
// belong to the same Redis primary; this is not a Redis Cluster API.
const publishCache = `if redis.call('GET', KEYS[1]) ~= ARGV[1] then return 0 end
if tonumber(ARGV[3]) > 0 then redis.call('SET', KEYS[2], ARGV[2], 'PX', ARGV[3]) else redis.call('SET', KEYS[2], ARGV[2]) end
return 1`

func (c *Cache[T]) store(ctx context.Context, key string, v T, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	owned, ok := ctx.Value(leaseContextKey{}).(lease)
	if !ok {
		return errors.New("singleflight: missing cache-fill lease")
	}
	n, err := c.client.Eval(ctx, publishCache, []string{owned.key, key}, owned.token, b, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if n != 1 {
		return lock.ErrExpired
	}
	return nil
}
