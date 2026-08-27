package ratelimit

import (
	"errors"
	"fmt"
	"time"
)

var ErrTooManyRequests = errors.New("too many requests")

type ratelimiter interface {
	Allow(key string) bool
	AllowN(key string, n int) bool
	Limit(key string) *Result
	LimitN(key string, n int) *Result
}

type Config struct {
	Limit  int
	Period time.Duration
	Burst  int
}

func (cfg Config) Validate() error {
	if cfg.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	if cfg.Period <= 0 {
		return errors.New("period must be greater than 0")
	}
	if cfg.Burst < 0 {
		return errors.New("burst must be equal or greater than 0")
	}

	return nil
}

type Result struct {
	Allow      bool
	Remaining  int
	ResetAfter time.Duration
	RetryAfter time.Duration
	Limit      int
}

func (r *Result) String() string {
	return fmt.Sprintf("allow=%t remaining=%d reset_after=%s retry_after=%s",
		r.Allow,
		r.Remaining,
		r.ResetAfter,
		r.RetryAfter,
	)
}
