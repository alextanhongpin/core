package singleflight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Cache[T any] struct {
	Client  *redis.Client
	Group   *Group
	LockTTL time.Duration
	WaitTTL time.Duration
	Suffix  string
}

func NewCache[T any](client *redis.Client) *Cache[T] {
	return &Cache[T]{
		Client:  client,
		Group:   New(client),
		LockTTL: 10 * time.Second,
		WaitTTL: 10 * time.Second,
		Suffix:  "fetch",
	}
}

func (c *Cache[T]) LoadOrStore(ctx context.Context, key string, getter func(ctx context.Context) (T, error), ttl time.Duration) (T, bool, error) {
	t, err := c.load(ctx, key)
	if err == nil {
		return t, true, nil
	}

	if !errors.Is(err, redis.Nil) {
		return t, false, err
	}

	did, err := c.Group.DoOrWait(ctx, fmt.Sprintf("%s:%s", key, c.Suffix), func(ctx context.Context) error {
		v, err := getter(ctx)
		if err != nil {
			return err
		}

		b, err := json.Marshal(v)
		if err != nil {
			return err
		}

		return c.Client.Set(ctx, key, b, ttl).Err()
	}, c.LockTTL, c.WaitTTL)
	if err != nil {
		return t, false, err
	}

	t, err = c.load(ctx, key)
	if err != nil {
		return t, false, err
	}

	return t, !did, nil
}

func (c *Cache[T]) load(ctx context.Context, key string) (t T, err error) {
	b, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(b, &t)
	return t, err
}
