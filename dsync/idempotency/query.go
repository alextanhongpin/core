package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type QueryOption[T comparable, V any] struct {
	LockTimeout     time.Duration
	RetentionPeriod time.Duration
	Handler         func(ctx context.Context, req T) (V, error)
}

type Query[T comparable, V any] struct {
	store     store[V]
	lock      time.Duration
	retention time.Duration
	handler   func(ctx context.Context, req T) (V, error)
}

func NewQuery[T comparable, V any](client *redis.Client, opt QueryOption[T, V]) *Query[T, V] {
	if opt.LockTimeout <= 0 {
		opt.LockTimeout = 1 * time.Minute
	}

	if opt.RetentionPeriod <= 0 {
		opt.RetentionPeriod = 24 * time.Hour
	}

	return &Query[T, V]{
		store:     newRedisStore[V](client),
		lock:      opt.LockTimeout,
		retention: opt.RetentionPeriod,
		handler:   opt.Handler,
	}
}

func (r *Query[T, V]) Query(ctx context.Context, key string, req T) (V, error) {
	// Sets the idempotency operation status to "started".
	// Can only be executed by one client.
	var v V
	ok, err := r.store.lock(ctx, key, r.lock)
	if err != nil {
		return v, err
	}

	// Started. Runs the idempotent operation and save the result.
	if ok {
		v, err = r.handler(ctx, req)
		if err != nil {
			// Delete the lock on fail.
			return v, errors.Join(err, r.store.unlock(ctx, key))
		}

		// Save the request/response pair on success.
		return v, r.save(ctx, key, req, v, r.retention)
	}

	// The lock is acquired. The request may be in flight, or already completed.
	return r.load(ctx, key, req)
}

func (r *Query[T, V]) save(ctx context.Context, key string, req T, res V, timeout time.Duration) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	d := data[V]{
		Status:   Success,
		Request:  hash(b),
		Response: res,
	}

	return r.store.save(ctx, key, d, timeout)
}

func (r *Query[T, V]) load(ctx context.Context, key string, req T) (V, error) {
	var v V
	d, err := r.store.load(ctx, key)
	if err != nil {
		return v, err
	}

	if d.Status == Started {
		return v, ErrRequestInFlight
	}

	b, err := json.Marshal(req)
	if err != nil {
		return v, err
	}

	if d.Request != hash(b) {
		return v, ErrRequestMismatch
	}

	return d.Response, nil
}
