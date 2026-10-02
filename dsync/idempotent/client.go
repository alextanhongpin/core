package idempotent

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
)

var _ client = (*Client)(nil)

// Client wraps a Redis client to provide atomic compare-and-swap,
// compare-and-delete, and load-or-store operations.
type Client struct {
	client *redis.Client
}

// NewClient returns a new Client wrapping the provided Redis client.
func NewClient(client *redis.Client) *Client {
	return &Client{
		client: client,
	}
}

// CompareAndSwap sets key to newValue if the current value in Redis matches oldValue.
// If the key does not exist or the value does not match, ErrLockConflict is returned.
func (c *Client) CompareAndSwap(ctx context.Context, key string, oldValue, newValue string, ttl time.Duration) error {
	_, err := c.client.SetIFDEQ(ctx, key, newValue, helper.DigestString(oldValue), ttl).Result()
	if errors.Is(err, redis.Nil) {
		return ErrLockConflict
	}
	return err
}

// CompareAndDelete deletes key from Redis if the current value matches oldValue.
// If the key does not exist or the value does not match, ErrLockConflict is returned.
func (c *Client) CompareAndDelete(ctx context.Context, key, oldValue string) error {
	n, err := c.client.DelExArgs(ctx, key, redis.DelExArgs{
		Mode:        "IFDEQ",
		MatchDigest: helper.DigestString(oldValue),
	}).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLockConflict
	}
	return nil
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// loaded is true if the value was loaded from Redis, or false if it was newly stored.
func (c *Client) LoadOrStore(ctx context.Context, key string, value string, ttl time.Duration) (curr string, loaded bool, err error) {
	result, err := c.client.SetArgs(ctx, key, value, redis.SetArgs{
		Get:  true,
		Mode: string(redis.NX),
		TTL:  ttl,
	}).Result()
	// If the key did not exist before SET NX, GET returns redis.Nil.
	// Since the value was successfully stored, treat this as a successful store.
	if errors.Is(err, redis.Nil) {
		return value, false, nil
	}
	if err != nil {
		return "", false, err
	}

	return result, true, nil
}
