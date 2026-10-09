package circuitbreaker

import (
	_ "embed"

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

func DefaultConfig() Config {
	return Config{
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

type Config struct {
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
	client *redis.Client
	cfg    Config
}

// Options is retained as a naming alias; constructors accept values.
type Options = Config

func (c Config) WithDefaults() Config {
	d := DefaultConfig()
	if c.FailureThreshold == 0 {
		c.FailureThreshold = d.FailureThreshold
	}
	if c.SuccessThreshold == 0 {
		c.SuccessThreshold = d.SuccessThreshold
	}
	if c.FailurePeriod == 0 {
		c.FailurePeriod = d.FailurePeriod
	}
	if c.SuccessPeriod == 0 {
		c.SuccessPeriod = d.SuccessPeriod
	}
	if c.OpenTimeout == 0 {
		c.OpenTimeout = d.OpenTimeout
	}
	if c.FailureCount == nil {
		c.FailureCount = d.FailureCount
	}
	if c.SlowCallCount == nil {
		c.SlowCallCount = d.SlowCallCount
	}
	return c
}
func (c Config) Validate() error {
	if c.FailureThreshold <= 0 || c.SuccessThreshold <= 0 {
		return errors.New("circuitbreaker: thresholds must be positive")
	}
	if c.FailurePeriod < time.Millisecond || c.SuccessPeriod < time.Millisecond || c.OpenTimeout < time.Millisecond {
		return errors.New("circuitbreaker: durations must be at least one millisecond")
	}
	return nil
}
func New(client *redis.Client, cfg Config) (*CircuitBreaker, error) {
	if client == nil {
		return nil, errors.New("circuitbreaker: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &CircuitBreaker{client: client, cfg: cfg}, nil
}
func MustNew(client *redis.Client, cfg Config) *CircuitBreaker {
	cb, err := New(client, cfg)
	if err != nil {
		panic(err)
	}
	return cb
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
		if cb.cfg.FailureCount != nil {
			failureCount += cb.cfg.FailureCount(cause)
		}
		if cb.cfg.SlowCallCount != nil {
			failureCount += cb.cfg.SlowCallCount(duration)
		}
	} else {
		successCount = 1
	}
	if failureCount < 0 {
		return Unknown, errors.New("circuitbreaker: negative failure weighting")
	}
	args := []any{failureCount, cb.cfg.FailureThreshold, cb.cfg.FailurePeriod.Milliseconds(), successCount, cb.cfg.SuccessThreshold, cb.cfg.SuccessPeriod.Milliseconds(), cb.cfg.OpenTimeout.Milliseconds(), generation}
	status, err := cb.client.FCall(ctx, method, []string{key}, args...).Int()
	return Status(status), err
}

func (cb *CircuitBreaker) SetStatus(ctx context.Context, key string, status Status) error {
	if status < Closed || status > ForcedOpen {
		return errors.New("circuitbreaker: invalid status")
	}
	return cb.client.FCall(ctx, "cb_set_status", []string{key}, status.Int(), cb.cfg.OpenTimeout.Milliseconds()).Err()
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
