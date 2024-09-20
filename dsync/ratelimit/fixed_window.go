package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

//go:embed fixed_window.lua
var fixedWindowScript string

var fixedWindow = redis.NewScript(fixedWindowScript)

// FixedWindow implements the Fixed Window algorithm.
type FixedWindow struct {
	client *redis.Client
	opt    *FixedWindowOption
	Now    func() time.Time
}

type FixedWindowOption struct {
	Limit  int64
	Period time.Duration
}

func (opt *FixedWindowOption) Valid() error {
	if opt.Limit <= 0 {
		return fmt.Errorf("%w: limit must be greater than 0", Error)
	}
	if opt.Period <= 0 {
		return fmt.Errorf("%w: period must be greater than 0", Error)
	}

	return nil
}

func NewFixedWindow(client *redis.Client, opt *FixedWindowOption) *FixedWindow {
	if opt == nil {
		panic("ratelimit: option is nil")
	}
	if err := opt.Valid(); err != nil {
		panic(err)
	}

	return &FixedWindow{
		client: client,
		opt:    opt,
		Now:    time.Now,
	}
}

func (r *FixedWindow) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	key = r.buildKey(r.Now(), key)
	now := r.Now()
	limit := r.opt.Limit
	period := r.opt.Period

	keys := []string{key}
	argv := []any{
		limit,
		now.UnixMilli(),
		period.Milliseconds(),
		n,
	}
	res, err := fixedWindow.Run(ctx, r.client, keys, argv...).Int64Slice()
	if err != nil {
		return nil, err
	}
	allow := res[0] == 1
	count := res[1]
	remaining := limit - count

	start := now.Truncate(period)
	end := start.Add(period)
	retryAt := now
	if remaining == 0 {
		retryAt = end
	}

	return &Result{
		Allow:     allow,
		Remaining: remaining,
		Limit:     limit,
		ResetAt:   end,
		RetryAt:   retryAt,
	}, nil
}

func (r *FixedWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return r.AllowN(ctx, key, 1)
}

func (r *FixedWindow) buildKey(now time.Time, key string) string {
	t := now.Truncate(r.opt.Period).Format(time.RFC3339Nano)
	// Set the key first to allow users to search their key by prefix.
	// The ratelimit:fixed_window is used to identify the
	// algorithm used, in case users switched the
	// implementation.
	return fmt.Sprintf("%s:ratelimit:fixed_window:%s", key, t)
}
