package ratelimit

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidNumber = errors.New("ratelimit: value must be positive")
)

func validate(limit int, period time.Duration, burst int) error {
	if limit <= 0 {
		return fmt.Errorf("%w: limit", ErrInvalidNumber)
	}
	if period <= 0 {
		return fmt.Errorf("%w: period", ErrInvalidNumber)
	}
	if burst < 0 {
		return fmt.Errorf("%w: burst", ErrInvalidNumber)
	}

	return nil
}

type Result struct {
	Allow      bool
	Remaining  int
	ResetAfter time.Duration
	RetryAfter time.Duration
}

func (r *Result) String() string {
	return fmt.Sprintf("allow=%t remaining=%d reset_after=%s retry_after=%s",
		r.Allow,
		r.Remaining,
		r.ResetAfter,
		r.RetryAfter,
	)
}
