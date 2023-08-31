// Package ratelimit implements distributed rate limiting using redis function.
package ratelimit

import (
	"context"
	_ "embed"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	gcraAlgorithm          = "gcra"
	slidingWindowAlgorithm = "sliding_window"
)

//go:embed ratelimit.lua
var ratelimit string

type RateLimiter struct {
	client *redis.Client
	limit  int
	every  time.Duration
	burst  int
}

type Option struct {
	Limit int
	Every time.Duration
	Burst int
}

func New(client *redis.Client, opt *Option) (*RateLimiter, error) {
	rl := &RateLimiter{
		client: client,
		limit:  opt.Limit,
		every:  opt.Every,
		burst:  opt.Burst,
	}

	return rl, rl.register()
}

type Result struct {
	Allow     bool
	Remaining int64
	RetryIn   time.Duration
	ResetIn   time.Duration
}

func newResult(res []int64) *Result {
	return &Result{
		Allow:     res[0] == 1,
		Remaining: res[1],
		RetryIn:   time.Duration(res[2]) * time.Millisecond,
		ResetIn:   time.Duration(res[3]) * time.Millisecond,
	}
}

// GCRA implements the Genetic Cell Rate algorithm.
func (r *RateLimiter) GCRA(ctx context.Context, key string) (*Result, error) {
	keys := []string{key}
	args := []any{r.limit, r.periodInMs(), r.burst}
	resp, err := r.client.FCall(ctx, gcraAlgorithm, keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

// SlidingWindow implements the sliding window algorithm.
func (r *RateLimiter) SlidingWindow(ctx context.Context, key string) (*Result, error) {
	keys := []string{key}
	args := []any{r.limit, r.periodInMs()}
	resp, err := r.client.FCall(ctx, slidingWindowAlgorithm, keys, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	return newResult(resp), nil
}

func (r *RateLimiter) register() error {
	_, err := r.client.FunctionLoadReplace(context.Background(), ratelimit).Result()
	if exists(err) {
		return nil
	}

	return err
}

func (r *RateLimiter) periodInMs() int64 {
	return r.every.Milliseconds()
}

func exists(err error) bool {
	// The ERR part is trimmed from prefix comparison.
	return redis.HasErrorPrefix(err, "Library 'ratelimit' already exists")
}
