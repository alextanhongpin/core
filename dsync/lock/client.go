package lock

import (
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/helper"
)

// Backend stores leases. Each method must be atomic and must compare token
// with the stored owner. Client is the Redis implementation; tests can supply
// a fake.
type Backend interface {
	// Lock creates key owned by token for ttl. It returns ErrLocked if key exists.
	Lock(ctx context.Context, key, token string, ttl time.Duration) error
	// Extend resets key's ttl if token still owns it, otherwise ErrLocked.
	Extend(ctx context.Context, key, token string, ttl time.Duration) error
	// Unlock deletes key if token still owns it, otherwise ErrExpired.
	Unlock(ctx context.Context, key, token string) error
}

// client is embedded in Locker under this name to keep the field unexported.
type client = Backend

var _ Backend = (*Client)(nil)

// Client is the Redis Backend. The key holds the owner's token as a plain
// string with a TTL. It needs Redis 8.4 or later for the DELEX and SET IFDEQ
// commands used by Unlock and Extend.
type Client struct {
	*redis.Client
}

// NewClient wraps a go-redis client. The caller keeps ownership of client and
// must close it after all lock operations finish.
func NewClient(client *redis.Client) *Client {
	return &Client{
		Client: client,
	}
}

// Lock runs SET key token NX PX ttl. It returns ErrLocked if key already exists.
func (c *Client) Lock(ctx context.Context, key, token string, ttl time.Duration) error {
	ok, err := c.Client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLocked
	}
	return nil
}

// Unlock deletes key only if it still holds token (DELEX key IFDEQ digest).
// It returns ErrExpired if the key is gone or owned by another token.
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

// Extend resets key's TTL only if it still holds token (SET key token IFDEQ
// digest PX ttl). It returns ErrLocked if the key is gone or owned by another
// token.
func (c *Client) Extend(ctx context.Context, key, token string, ttl time.Duration) error {
	val := []byte(token)
	err := c.Client.SetIFDEQ(ctx, key, val, helper.DigestString(token), ttl).Err()
	if errors.Is(err, redis.Nil) {
		return ErrLocked
	}
	return err
}
