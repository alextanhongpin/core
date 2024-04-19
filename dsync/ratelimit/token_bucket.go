package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed token_bucket.lua
var tokenBucketScript string

var tokenBucket = redis.NewScript(tokenBucketScript)

type TokenBucketOption struct {
	Limit  int64
	Period time.Duration
	Burst  int64
}

type TokenBucket struct {
	client *redis.Client
	opt    *TokenBucketOption
	Now    func() time.Time
}

func NewTokenBucket(client *redis.Client, opt *TokenBucketOption) *TokenBucket {
	if opt == nil {
		panic("ratelimit: option is nil")
	}
	if opt.Limit <= 0 {
		panic("ratelimit: limit is invalid")
	}
	if opt.Period <= 0 {
		panic("ratelimit: period is invalid")
	}

	return &TokenBucket{
		client: client,
		opt:    opt,
		Now:    time.Now,
	}
}

func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int) (*Result, error) {
	now := tb.Now()
	keys := []string{tb.buildKey(now, key)}
	argv := []any{
		tb.opt.Limit,
		tb.opt.Period.Milliseconds(),
		tb.opt.Burst,
		now.UnixNano() / 1e6,
		n,
	}
	res, err := tokenBucket.Run(ctx, tb.client, keys, argv...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(res, tb.opt.Limit), nil
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return tb.AllowN(ctx, key, 1)
}

func (tb *TokenBucket) buildKey(now time.Time, key string) string {
	t := now.Truncate(tb.opt.Period).Format(time.RFC3339Nano)
	// Set the key first to allow users to search their key by prefix.
	// The ratelimit:fixed_window is used to identify the
	// algorithm used, in case users switched the
	// implementation.
	return fmt.Sprintf("%s:ratelimit:token_bucket:%s", key, t)
}
