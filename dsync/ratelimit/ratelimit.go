// Package ratelimit implements distributed rate limiting using redis function.
package ratelimit

import (
	"time"
)

type Result struct {
	Allow     bool
	Remaining int64
	RetryAt   time.Time
	ResetAt   time.Time
	Limit     int64
}

func newResult(res []int64, limit int64) *Result {
	return &Result{
		Allow:     res[0] == 1,
		Remaining: res[1],
		RetryAt:   unixMillisecondToTime(res[2]),
		ResetAt:   unixMillisecondToTime(res[3]),
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

func unixMillisecondToTime(unixMs int64) time.Time {
	ns := unixMs * 1e6
	return time.Unix(0, ns)
}
