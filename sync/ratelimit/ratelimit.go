package ratelimit

import "time"

type Result struct {
	Allow     bool
	Limit     int64
	Remaining int64
	RetryAt   time.Time
	ResetAt   time.Time
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
