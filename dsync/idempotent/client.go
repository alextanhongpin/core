package idempotent

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
)

var _ client = (*Client)(nil)

type Client struct {
	client *redis.Client
}

func NewClient(client *redis.Client) *Client {
	return &Client{
		client: client,
	}
}

// compareAndSwap swaps the old and new values for key if the value stored in
// the map is equal to old. The old value must be of a comparable type.
func (c *Client) CompareAndSwap(ctx context.Context, key string, oldValue, newValue string, ttl time.Duration) error {
	_, err := c.client.SetIFDEQ(ctx, key, newValue, helper.DigestString(oldValue), ttl).Result()
	if errors.Is(err, redis.Nil) {
		return ErrLockConflict
	}
	return err
}

// CompareAndDelete deletes the entry for key if its value is equal to old. The
// old value must be of a comparable type.
// If there is no current value for key in the map, CompareAndDelete returns
// false (even if the old value is the nil interface value).
func (r *Client) CompareAndDelete(ctx context.Context, key, oldValue string) error {
	n, err := r.client.DelExArgs(ctx, key, redis.DelExArgs{
		Mode:        "IFDEQ",
		MatchDigest: helper.DigestString(oldValue),
	}).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRequestMismatch
	}
	return nil
}

// loadOrStore returns the existing value for the key if present. Otherwise, it
// stores and returns the given value. The loaded result is true if the value
// was loaded, false if stored.
// Also see usecase here: https://github.com/golang/go/issues/33762#issuecomment-523757434
func (r *Client) LoadOrStore(ctx context.Context, key string, value string, ttl time.Duration) (curr string, loaded bool, err error) {
	c, err := r.client.SetArgs(ctx, key, value, redis.SetArgs{
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
		return "", false, err
	}

	return c, true, nil
}
