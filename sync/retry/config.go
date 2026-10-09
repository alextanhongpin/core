package retry

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

// Config controls a logical operation. Dependencies and callbacks are shared;
// they must support concurrent calls when the Retry is used concurrently.
type Config struct {
	// MaxRetries excludes the initial call. Zero disables retries.
	MaxRetries int
	Backoff    Backoff
	// Retryable returns the classified error and whether to retry it.
	// A nil policy retries all errors except cancellation and deadline errors.
	Retryable func(error) (error, bool)
	Throttler Limiter
}

// DefaultRetryable retries errors other than cancellation and deadline errors.
func DefaultRetryable(err error) (error, bool) {
	return err, !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

// NonRetryableErrors preserves err and rejects it if any target matches through
// wrapping or joining. The targets are copied so later slice changes are safe.
func NonRetryableErrors(targets ...error) func(error) (error, bool) {
	targets = slices.Clone(targets)
	return func(err error) (error, bool) {
		for _, target := range targets {
			if errors.Is(err, target) {
				return err, false
			}
		}
		return err, true
	}
}

// DefaultConfig enables ten retries with jitter and a shared adaptive budget.
func DefaultConfig() Config {
	return Config{MaxRetries: 10}.WithDefaults()
}

// WithDefaults fills omitted dependencies without changing MaxRetries.
func (c Config) WithDefaults() Config {
	if c.Backoff == nil {
		c.Backoff = NewExponentialBackoff(time.Second, time.Minute)
	}
	if c.Throttler == nil {
		c.Throttler, _ = NewThrottler(DefaultThrottlerConfig())
	}
	if c.Retryable == nil {
		c.Retryable = DefaultRetryable
	}
	return c
}

// Validate checks configuration without mutating it.
func (c Config) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("retry: max retries must be nonnegative")
	}
	return nil
}
