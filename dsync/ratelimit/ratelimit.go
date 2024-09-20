// Package ratelimit implements distributed rate limiting using redis function.
package ratelimit

import (
	"errors"
	"time"
)

var Error = errors.New("ratelimit")

type Result struct {
	Allow     bool
	Limit     int64
	Remaining int64
	ResetAt   time.Time
	RetryAt   time.Time
}

func newResult(res []int64, limit int64) *Result {
	return &Result{
		Allow:     res[0] == 1,
		Remaining: res[1],
		RetryAt:   time.UnixMilli(res[2]),
		ResetAt:   time.UnixMilli(res[3]),
		Limit:     limit,
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
