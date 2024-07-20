package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// SingleFlight is a policy for handling concurrent requests.
// When enabled, it will prevent multiple requests from hitting the same key.
// If the key is not found, it will execute the getter function and cache the result.
type SingleFlight struct {
	// If true,then it will not wait for the cache to be populated, but instead
	// it will hit the getter immediately.
	Passthrough bool
	// When non-empty, it will wait for the duration before attempting to request
	// the key from the cache again.
	Retries []time.Duration
	// Allows customizing the key that is used to lock.
	KeyFn func(key string) string
	// How long to lock the key.
	Lock time.Duration
}

type Cacheable[T any] struct {
	client       *redis.Client
	getter       func(ctx context.Context) (T, error)
	SingleFlight *SingleFlight
}

func New[T any](client *redis.Client, getter func(ctx context.Context) (T, error)) *Cacheable[T] {
	return &Cacheable[T]{
		client: client,
		getter: getter,
	}
}

func (c *Cacheable[T]) Get(ctx context.Context, key string, ttl time.Duration) (v T, hit bool, err error) {
	v, err = c.get(ctx, key)
	if err == nil {
		return v, true, nil
	}

	if !errors.Is(err, redis.Nil) {
		return v, false, err
	}

	if c.SingleFlight != nil {
		sf := c.SingleFlight
		var v T
		// Lock the key.
		ok, err := c.client.SetNX(ctx, sf.KeyFn(key), fmt.Sprint(time.Now().Unix()), sf.Lock).Result()
		if err != nil {
			return v, false, err
		}

		// Fail to acquire the lock, another operation is filling the cache.
		if !ok && !sf.Passthrough {
			// Periodically check the cache.
			for _, d := range sf.Retries {
				time.Sleep(d)

				// If the cache is filled, return the value.
				v, err := c.get(ctx, key)
				if err == nil {
					return v, true, nil
				}
			}
			// Otherwise, hit the getter.
		}
	}

	v, err = c.getter(ctx)
	if err != nil {
		return v, false, err
	}

	if err := c.Set(ctx, key, v, ttl); err != nil {
		return v, false, err
	}

	return v, false, nil
}

func (c *Cacheable[T]) Set(ctx context.Context, key string, v T, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, string(b), ttl).Err()
}

func (c *Cacheable[T]) get(ctx context.Context, key string) (T, error) {
	var v T
	b, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, err
	}
	return v, nil
}
