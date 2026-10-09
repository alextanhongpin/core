// Package retry executes operations with backoff and a shared retry budget.
package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrLimitExceeded = errors.New("retry: limit exceeded")
	ErrThrottled     = errors.New("retry: throttled")
	ErrCanceled      = errors.New("retry: canceled")
)

// Runner executes callbacks synchronously and sequentially, completing them
// before returning. It may reject without calling the callback. Implementations
// must propagate callback errors unless their documented policy handles them.
type Runner interface {
	Do(context.Context, func(context.Context) error) error
}

// Func returns the last callback result and the runner's final error. If the
// runner rejects before invoking fn, the result is zero. Callers own cleanup of
// resource-bearing results from every attempt, including failed attempts.
// Concurrent invocations are safe if fn, runner, and shared inputs are safe.
func Func[K, V any](fn func(context.Context, K) (V, error), runner Runner) func(context.Context, K) (V, error) {
	return func(ctx context.Context, req K) (V, error) {
		var res V
		err := runner.Do(ctx, func(attemptCtx context.Context) error {
			var err error
			res, err = fn(attemptCtx, req)
			return err
		})
		return res, err
	}
}

// Retry owns its configuration snapshot. Dependencies remain shared. A Retry
// supports concurrent operations if its configured callbacks and dependencies do.
// Construct Retry with New; its zero value is not usable.
type Retry struct{ cfg Config }

// New defaults and validates a configuration copy. Config{} performs one call.
func New(cfg Config) (*Retry, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Retry{cfg: cfg}, nil
}

func canceled(ctx context.Context) error {
	if ctx.Err() == nil {
		return nil
	}
	return errors.Join(ErrCanceled, ctx.Err(), context.Cause(ctx))
}

// Do performs at most MaxRetries+1 sequential calls. Backoff.At(1) is the
// first retry. Only retries consume tokens; successful calls replenish them.
// Cancellation is checked before admission and invocation, but cancellation
// racing with invocation remains cooperative. A successful callback returns nil.
// Terminal budget errors wrap the last classified error, not the full history.
func (r *Retry) Do(ctx context.Context, fn func(context.Context) error) error {
	var last error
	for attempt := 0; ; attempt++ {
		if err := canceled(ctx); err != nil {
			return err
		}
		if attempt > 0 {
			if !r.cfg.Throttler.Allow() {
				return errors.Join(last, ErrThrottled)
			}
			timer := time.NewTimer(r.cfg.Backoff.At(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return canceled(ctx)
			case <-timer.C:
			}
			if err := canceled(ctx); err != nil {
				return err
			}
		}
		err := fn(ctx)
		if err == nil {
			r.cfg.Throttler.Success()
			return nil
		}
		if cancelErr := canceled(ctx); cancelErr != nil {
			return cancelErr
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return errors.Join(ErrCanceled, err)
		}
		classified, again := r.cfg.Retryable(err)
		if classified == nil {
			classified = err
		}
		if errors.Is(classified, context.Canceled) || errors.Is(classified, context.DeadlineExceeded) {
			return errors.Join(ErrCanceled, classified)
		}
		if !again {
			return classified
		}
		last = classified
		if attempt == r.cfg.MaxRetries {
			return errors.Join(last, fmt.Errorf("%w: retried %d times", ErrLimitExceeded, r.cfg.MaxRetries))
		}
	}
}
