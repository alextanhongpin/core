package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// Status is the state of a breaker. The numeric values are stored in Redis
// and must not change.
type Status int

// Int returns the numeric value of s as stored in Redis.
func (s Status) Int() int {
	return int(s)
}

// String returns a lowercase name such as "closed" or "half-open".
func (s Status) String() string {
	if text, ok := statusText[s]; ok {
		return text
	}
	return fmt.Sprintf("Status(%d)", int(s))
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
	// Unknown is the zero value; it is never stored.
	Unknown Status = iota
	// Closed lets calls through and counts failures. A key that does not
	// exist in Redis is Closed.
	Closed
	// HalfOpen lets calls through as probes after OpenTimeout. One failure
	// reopens the breaker; SuccessThreshold successes close it.
	HalfOpen
	// Opened rejects calls with ErrOpened until OpenTimeout has passed.
	Opened
	// Disabled lets every call through without counting. Set it with
	// SetStatus; it stays until changed.
	Disabled
	// ForcedOpen rejects every call with ErrOpened. Set it with SetStatus;
	// it stays until changed (OpenTimeout does not apply).
	ForcedOpen
)

// ErrOpened is returned by Do when the breaker is Opened or ForcedOpen.
// The function was not called. Fail fast or serve a fallback.
var ErrOpened = errors.New("cb: opened")

// commitTimeout bounds the Redis call that records a result after the
// operation ran. It is detached from the caller's context so that a request
// cancelled by the operation's own timeout is still counted.
const commitTimeout = 5 * time.Second

// DefaultConfig returns the configuration used for zero fields of Config:
//
//   - FailureThreshold 100, FailurePeriod 1s
//   - SuccessThreshold 20, SuccessPeriod 1s
//   - OpenTimeout 1m
//   - FailureCount: +2 for context.DeadlineExceeded (a timeout counts as 3)
//   - SlowCallCount: +1 for failed calls slower than 1s, +2 from 30s, +4 from 1m
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

// Config configures a CircuitBreaker. Zero fields take the values from
// DefaultConfig. Config is copied by New, so changing it afterwards has no
// effect on an existing breaker.
type Config struct {
	// FailureThreshold is the failure weight at which a Closed breaker opens.
	FailureThreshold int

	// FailurePeriod is how long the failure count is kept after the most
	// recent failure. Each failure restarts it, so the breaker opens after
	// FailureThreshold failures with no gap longer than FailurePeriod.
	// Must be at least 1ms.
	FailurePeriod time.Duration

	// SuccessThreshold is the number of successful probes at which a
	// HalfOpen breaker closes.
	SuccessThreshold int

	// SuccessPeriod is how long the success count is kept after the most
	// recent success in HalfOpen. Must be at least 1ms.
	SuccessPeriod time.Duration

	// OpenTimeout is how long the breaker stays Opened before letting probes
	// through. Must be at least 1ms.
	OpenTimeout time.Duration

	// FailureCount returns extra weight for a failed call, added to the base
	// weight of 1. It must be safe for concurrent use and must not return a
	// negative number. Return 0 to count every error once.
	FailureCount func(cause error) int

	// SlowCallCount returns extra weight for a failed call based on how long
	// it took. Successful calls are never counted as failures, however slow.
	// Same rules as FailureCount.
	SlowCallCount func(duration time.Duration) int
}

// Options is an alias of Config kept for backward compatibility.
//
// Deprecated: use Config.
type Options = Config

// CircuitBreaker runs functions through a breaker whose state is stored in
// Redis, one hash per key. It is safe for concurrent use.
type CircuitBreaker struct {
	client redis.UniversalClient
	cfg    Config
}

// WithDefaults returns a copy of c with zero fields replaced by the values
// from DefaultConfig.
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

// Validate reports whether c is usable as is. It does not apply defaults,
// so call WithDefaults first if c may contain zero fields. New does both.
func (c Config) Validate() error {
	if c.FailureThreshold <= 0 || c.SuccessThreshold <= 0 {
		return errors.New("circuitbreaker: thresholds must be positive")
	}
	if c.FailurePeriod < time.Millisecond || c.SuccessPeriod < time.Millisecond || c.OpenTimeout < time.Millisecond {
		return errors.New("circuitbreaker: durations must be at least one millisecond")
	}
	return nil
}

// New returns a CircuitBreaker. Zero config fields take the defaults from
// DefaultConfig; invalid values return an error.
//
// The client may be a *redis.Client, *redis.ClusterClient or any other
// redis.UniversalClient. It is not closed by the breaker. Run Setup once
// before using the breaker.
func New(client redis.UniversalClient, cfg Config) (*CircuitBreaker, error) {
	if isNil(client) {
		return nil, errors.New("circuitbreaker: nil client")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &CircuitBreaker{client: client, cfg: cfg}, nil
}

// MustNew is like New but panics on error. Use it for startup configuration
// that is known to be valid.
func MustNew(client redis.UniversalClient, cfg Config) *CircuitBreaker {
	cb, err := New(client, cfg)
	if err != nil {
		panic(err)
	}
	return cb
}

// Do runs fn through the breaker identified by key and returns fn's error.
//
//   - Opened or ForcedOpen: fn is not called and Do returns ErrOpened.
//   - Disabled: fn is called and nothing is recorded.
//   - Closed or HalfOpen: fn is called and its result is recorded.
//
// Every non-nil error from fn counts as a failure; filter errors that are
// not the dependency's fault (such as validation errors) out of fn if they
// should not trip the breaker.
//
// Do costs one Redis call, plus a second one after a failure or a HalfOpen
// probe. If Redis is unreachable before fn runs, Do returns the Redis error
// without calling fn. If recording a failure fails, the Redis error is
// joined with fn's error. If recording a success fails, Do still returns nil,
// so that a call which did happen is never reported as failed.
//
// fn is not limited to one concurrent caller, not even in HalfOpen.
func (cb *CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	admission, err := cb.client.FCall(ctx, "cb_begin", []string{key}).Int64Slice()
	if err != nil {
		return wrapErr(err)
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
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commitTimeout)
	defer cancel()
	_, commitErr := cb.commit(commitCtx, key, operationErr, time.Since(start), generation)
	if operationErr == nil {
		// The probe succeeded; losing one success only delays closing.
		return nil
	}
	return errors.Join(operationErr, commitErr)
}

func (cb *CircuitBreaker) commit(ctx context.Context, key string, cause error, duration time.Duration, generation int64) (Status, error) {
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
	status, err := cb.client.FCall(ctx, "cb_commit", []string{key}, args...).Int()
	return Status(status), wrapErr(err)
}

// SetStatus moves the breaker for key to status and resets its counters.
// Use it for manual control, for example ForcedOpen during an incident,
// Disabled to bypass the breaker, or Closed to reset it. Setting Opened
// starts a new OpenTimeout. Results of calls that started before the change
// are ignored.
func (cb *CircuitBreaker) SetStatus(ctx context.Context, key string, status Status) error {
	if status < Closed || status > ForcedOpen {
		return errors.New("circuitbreaker: invalid status")
	}
	return wrapErr(cb.client.FCall(ctx, "cb_set_status", []string{key}, status.Int(), cb.cfg.OpenTimeout.Milliseconds()).Err())
}

// Status returns the stored status for key; a key that does not exist is
// Closed. An Opened breaker whose OpenTimeout has passed is still reported
// as Opened until the next Do moves it to HalfOpen.
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
