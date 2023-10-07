// Package ratelimit implements distributed rate limiting using redis function.
package ratelimit

import (
	"context"
	_ "embed"
	"time"

	redis "github.com/redis/go-redis/v9"
)

//go:embed ratelimit.lua
var ratelimit string

type Result struct {
	Allow     bool
	Remaining int64
	RetryAt   time.Time
	ResetAt   time.Time
}

func newResult(res []int64) *Result {
	return &Result{
		Allow:     res[0] == 1,
		Remaining: res[1],
		RetryAt:   unixMillisecondToTime(res[2]),
		ResetAt:   unixMillisecondToTime(res[3]),
	}
}

func (r *Result) RetryIn() time.Duration {
	retryIn := r.RetryAt.Sub(time.Now())
	if retryIn < 0 {
		return 0
	}

	return retryIn
}

func (r *Result) ResetIn() time.Duration {
	resetIn := r.ResetAt.Sub(time.Now())
	if resetIn < 0 {
		return 0
	}

	return resetIn
}

func (r *Result) Wait() {
	time.Sleep(r.RetryIn())
}

func unixMillisecondToTime(unixMs int64) time.Time {
	sec := unixMs / 1e3
	ns := unixMs % 1e3 * 1e6
	return time.Unix(sec, ns)
}

func registerFunction(client *redis.Client) {
	_, err := client.FunctionLoadReplace(context.Background(), ratelimit).Result()
	if err != nil {
		if exists(err) {
			return
		}

		panic(err)
	}
}

func exists(err error) bool {
	// The ERR part is trimmed from prefix comparison.
	return redis.HasErrorPrefix(err, "Library 'ratelimit' already exists")
}
