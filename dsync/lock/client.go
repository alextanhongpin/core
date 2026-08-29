package lock

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
)

type client interface {
	Lock(ctx context.Context, key, token string, ttl, wait time.Duration) error
	Unlock(ctx context.Context, key, token string) error
	Extend(ctx context.Context, key, token string, ttl time.Duration) error
}

var _ client = (*Client)(nil)

type Client struct {
	*redis.Client
}

func NewClient(client *redis.Client) *Client {
	return &Client{
		Client: client,
	}
}

func (c *Client) Lock(ctx context.Context, key, token string, ttl, wait time.Duration) error {
	if wait <= 0 {
		return c.tryLock(ctx, key, token, ttl)
	}

	return c.lockWait(ctx, key, token, ttl, wait)
}

// Unlocks the key with the given token.
func (c *Client) Unlock(ctx context.Context, key, token string) error {
	n, err := c.Client.DelExArgs(ctx, key, redis.DelExArgs{
		Mode:        "IFDEQ",
		MatchDigest: helper.DigestString(token),
	}).Result()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrExpired
	}
	return nil
}

func (c *Client) Extend(ctx context.Context, key, token string, ttl time.Duration) error {
	val := []byte(token)
	err := c.Client.SetIFDEQ(ctx, key, val, helper.DigestString(token), ttl).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}
	return err
}

// lockWait waits until the lock is acquired.
func (c *Client) lockWait(ctx context.Context, key, token string, ttl, wait time.Duration) error {
	// NOTE: We don't use context for cancellation because it will be passed down.
	timeout := time.After(wait)
	tryLock := func() error {
		return c.tryLock(ctx, key, token, ttl)
	}

	var sleep time.Duration
	for {
		select {
		case <-timeout:
			err := tryLock()
			if errors.Is(err, ErrLocked) {
				return ErrLockWaitTimeout
			}

			return err
		case <-ctx.Done():
			return context.Cause(ctx)

		case <-time.After(sleep):
			err := tryLock()
			if errors.Is(err, ErrLocked) {
				sleep = rand.N(wait)

				continue
			}

			return err
		}
	}
}

func (c *Client) tryLock(ctx context.Context, key, token string, ttl time.Duration) error {
	err := c.Client.SetArgs(ctx, key, token, redis.SetArgs{
		Mode: string(redis.NX),
		TTL:  ttl,
	}).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}

	return err
}
