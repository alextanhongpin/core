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

func (s Status) String() string {
	return statusText[s]
}

var statusText = map[Status]string{
	Unknown:    "unknown",
	Closed:     "closed",
	HalfOpen:   "half-open",
	Opened:     "opened",
	Disabled:   "disabled",
	ForcedOpen: "forced-open",
}

const (
	Unknown Status = iota
	Closed
	HalfOpen
	Opened
	Disabled
	ForcedOpen
)

var ErrOpened = errors.New("cb: opened")

func DefaultConfig() *Options {
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
		options: cmp.Or(opts, DefaultConfig()),
	}
}

func (cb *CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	admission, err := cb.client.FCall(ctx, "cb_begin", []string{key}).Int64Slice()
	if err != nil {
		return err
	}
	if len(admission) != 2 {
		return errors.New("circuitbreaker: invalid admission response")
	}
	status, generation := Status(admission[0]), admission[1]
	switch status {
	case Opened, ForcedOpen:
		return ErrOpened
	case Disabled:
		return fn()
	case Closed, HalfOpen:
	default:
		return errors.New("circuitbreaker: invalid state")
	}
	start := time.Now()
	operationErr := fn()
	if operationErr == nil && status == Closed {
		return nil
	}
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, commitErr := cb.call(commitCtx, "cb_commit", key, operationErr, time.Since(start), generation)
	return errors.Join(operationErr, commitErr)
}

func (cb *CircuitBreaker) call(ctx context.Context, method, key string, cause error, duration time.Duration, generation int64) (Status, error) {
	failureCount, successCount := 0, 0
	if cause != nil {
		failureCount = 1
		if cb.options.FailureCount != nil {
			failureCount += cb.options.FailureCount(cause)
		}
		if cb.options.SlowCallCount != nil {
			failureCount += cb.options.SlowCallCount(duration)
		}
	} else {
		successCount = 1
	}
	if failureCount < 0 {
		return Unknown, errors.New("circuitbreaker: negative failure weighting")
	}
	args := []any{failureCount, cb.options.FailureThreshold, cb.options.FailurePeriod.Milliseconds(), successCount, cb.options.SuccessThreshold, cb.options.SuccessPeriod.Milliseconds(), cb.options.OpenTimeout.Milliseconds(), generation}
	status, err := cb.client.FCall(ctx, method, []string{key}, args...).Int()
	return Status(status), err
}

func (cb *CircuitBreaker) SetStatus(ctx context.Context, key string, status Status) error {
	if status < Closed || status > ForcedOpen {
		return errors.New("circuitbreaker: invalid status")
	}
	return cb.client.FCall(ctx, "cb_set_status", []string{key}, status.Int(), cb.options.OpenTimeout.Milliseconds()).Err()
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
