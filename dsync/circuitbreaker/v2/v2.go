package v2

import (
	_ "embed"
	"errors"

	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	Opened     = 1
	HalfOpen   = 0
	Closed     = -1
	Disabled   = -3
	ForcedOpen = -2
	Unknown    = 99
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
	status, err := cb.begin(ctx, key)
	if err != nil {
		return err
	}
	switch status {
	case HalfOpen:
	case Closed:
	case Disabled:
		return fn()
	case ForcedOpen:
		return ErrOpened
	case Unknown:
		panic("unknown status")
	default:
		return ErrOpened
	}

	// Do not pass context, as the cancelation should not affect the redis cancelation.
	err = fn()
	if err != nil {
		if _, err := cb.commit(ctx, key, err); err != nil {
			return err
		}
		return err
	}

	if status == Closed {
		return nil
	}
	switch status {
	case Closed:
		return nil
	}
	_, err = cb.commit(ctx, key, err)
	return err
}

func (cb *CircuitBreaker) begin(ctx context.Context, key string) (int, error) {
	var failureCount, successCount int
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

	status, err := cb.client.FCall(ctx, "begin", keys, args...).Int()
	return status, err
}

func (cb *CircuitBreaker) commit(ctx context.Context, key string, cause error) (int, error) {
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
	status, err := cb.client.FCall(ctx, "commit", keys, args...).Int()
	return status, err
}
