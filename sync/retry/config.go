package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Config struct {
	Attempts  int
	Backoff   Backoff
	Retryable func(err error) (error, bool)
	Throttler Limiter
}

var DefaultRetryable = NonRetryableErrors(context.Canceled, context.DeadlineExceeded)

func NonRetryableErrors(errs ...error) func(error) (error, bool) {
	joinErr := errors.Join(errs...)
	return func(err error) (error, bool) {
		if errors.Is(joinErr, err) {
			return fmt.Errorf("%w: %w", ErrCanceled, err), false
		}
		return err, true
	}
}

func DefaultConfig() *Config {
	return &Config{
		Attempts:  10,
		Backoff:   NewExponentialBackoff(time.Second, time.Minute),
		Throttler: NewThrottler(DefaultThrottlerConfig()),
		Retryable: DefaultRetryable,
	}
}
