package circuitbreaker

import (
	_ "embed"

	"cmp"
	"context"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Status int

func (s Status) Int() int {
	return int(s)
}

const (
	Unknown    Status = 0
	Closed     Status = 1
	HalfOpen   Status = 2
	Opened     Status = 3
	Disabled   Status = 4
	ForcedOpen Status = 5
)

var ErrOpened = errors.New("cb: opened")

func NewOptions() *Options {
	return &Options{
		FailureThreshold: 100,
		FailurePeriod:    time.Second,
		SuccessThreshold: 20,
		SuccessPeriod:    time.Second,
		OpenTimeout:      time.Minute,
		FailureCount: func(cause error) int {
			if errors.Is(cause, context.DeadlineExceeded) {
				return 2
			}
			return 0
		},
		SlowCallCount: func(duration time.Duration) int {
			if duration >= time.Minute {
				return 4
			}
			if duration >= 30*time.Second {
				return 2
			}
			if duration > time.Second {
				return 1
			}

			return 0
		},
	}
}

type Options struct {
	FailureThreshold int
	FailurePeriod    time.Duration
	SuccessThreshold int
	SuccessPeriod    time.Duration
	OpenTimeout      time.Duration
	FailureCount     func(cause error) int
	SlowCallCount    func(duration time.Duration) int
}

// CircuitBreaker ...
type CircuitBreaker struct {
	client  *redis.Client
	options *Options
}

func New(client *redis.Client, opts *Options) *CircuitBreaker {
	return &CircuitBreaker{
		client:  client,
		options: cmp.Or(opts, NewOptions()),
	}
}

func (cb *CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error {
	status, err := cb.call(ctx, "begin", key, nil, 0)
	if err != nil {
		return err
	}
	switch status {
	case Closed, HalfOpen:
	case Opened, ForcedOpen:
		return ErrOpened
	case Disabled:
		return fn()
	default:
		panic("unknown status")
	}

	start := time.Now()
	// Do not pass context, as the cancelation should not affect the redis cancelation.
	err = fn()
	if err != nil {
		_, callErr := cb.call(ctx, "commit", key, err, time.Since(start))
		return cmp.Or(callErr, err)
	}
	if status != HalfOpen {
		return nil
	}

	_, err = cb.call(ctx, "commit", key, nil, 0)
	return err
}

func (cb *CircuitBreaker) call(ctx context.Context, method, key string, cause error, duration time.Duration) (Status, error) {
	var failureCount int
	var successCount int
	if cause != nil {
		failureCount = 1 + cb.options.FailureCount(cause) + cb.options.SlowCallCount(duration)
	} else {
		successCount = 1
	}

	keys := []string{key}
	args := []any{
		failureCount,
		cb.options.FailureThreshold,
		cb.options.FailurePeriod.Milliseconds(),
		successCount,
		cb.options.SuccessThreshold,
		cb.options.SuccessPeriod.Milliseconds(),
		cb.options.OpenTimeout.Milliseconds(),
	}
	status, err := cb.client.FCall(ctx, method, keys, args...).Int()
	return Status(status), err
}

func (cb *CircuitBreaker) SetStatus(ctx context.Context, key string, status Status) error {
	_, err := cb.client.HSet(ctx, key, "status", status.Int()).Result()
	return err
}

func (cb *CircuitBreaker) Status(ctx context.Context, key string) (Status, error) {
	n, err := cb.client.HGet(ctx, key, "status").Int()
	if errors.Is(err, redis.Nil) {
		return Closed, nil
	}

	if err != nil {
		return 0, err
	}

	return Status(n), nil
}
