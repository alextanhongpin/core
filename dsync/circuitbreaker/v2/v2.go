package v2

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

// CircuitBreaker ...
type CircuitBreaker struct {
	client           *redis.Client
	failureThreshold int
	failurePeriod    time.Duration
	successThreshold int
	successPeriod    time.Duration
	openTimeout      time.Duration
}

func NewCircuitBreaker(client *redis.Client) *CircuitBreaker {
	return &CircuitBreaker{
		client:           client,
		failureThreshold: 10,
		failurePeriod:    time.Second,
		successThreshold: 10,
		successPeriod:    10,
		openTimeout:      time.Minute,
	}
}

func (cb *CircuitBreaker) Do(ctx context.Context, key string, fn func() error) error {
	status, err := cb.call(ctx, "begin", key, nil)
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

	// Do not pass context, as the cancelation should not affect the redis cancelation.
	err = fn()
	if err != nil {
		_, callErr := cb.call(ctx, "commit", key, err)
		return cmp.Or(callErr, err)
	}
	if status != HalfOpen {
		return nil
	}

	_, err = cb.call(ctx, "commit", key, err)
	return err
}

func (cb *CircuitBreaker) call(ctx context.Context, method, key string, cause error) (Status, error) {
	var failureCount int
	var successCount int
	if cause != nil {
		failureCount = 1 // + cb.failureCountMultiplier(cause)
	} else {
		successCount = 1
	}

	keys := []string{key}
	args := []any{
		failureCount,
		cb.failureThreshold,
		cb.failurePeriod.Milliseconds(),
		successCount,
		cb.successThreshold,
		cb.successPeriod.Milliseconds(),
		cb.openTimeout.Milliseconds(),
	}
	status, err := cb.client.FCall(ctx, method, keys, args...).Int()
	return Status(status), err
}

func (cb *CircuitBreaker) SetStatus(ctx context.Context, key string, status Status) error {
	keys := []string{key}
	args := []any{status.Int()}
	_, err := cb.client.FCall(ctx, "set_status", keys, args...).Int()
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
