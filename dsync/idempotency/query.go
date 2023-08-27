package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type query[T, U any] interface {
	Query(ctx context.Context, req T) (res U, err error)
}

type QueryHandler[T, U any] func(ctx context.Context, req T) (res U, err error)

func (h QueryHandler[T, U]) Query(ctx context.Context, req T) (U, error) {
	return h(ctx, req)
}

type QueryOption[T comparable, U any] struct {
	LockTimeout     time.Duration
	RetentionPeriod time.Duration
	Handler         query[T, U]
}

type Query[T comparable, U any] struct {
	store     store[U]
	lock      time.Duration
	retention time.Duration
	handler   query[T, U]
}

func NewQuery[T comparable, U any](store store[U], opt QueryOption[T, U]) *Query[T, U] {
	if opt.LockTimeout <= 0 {
		opt.LockTimeout = 1 * time.Minute
	}

	if opt.RetentionPeriod <= 0 {
		opt.RetentionPeriod = 24 * time.Hour
	}

	return &Query[T, U]{
		store:     store,
		lock:      opt.LockTimeout,
		retention: opt.RetentionPeriod,
		handler:   opt.Handler,
	}
}

func (r *Query[T, U]) Query(ctx context.Context, key string, req T) (U, error) {
	// Sets the idempotency operation status to "started".
	// Can only be executed by one client.
	var v U
	ok, err := r.store.Lock(ctx, key, r.lock)
	if err != nil {
		return v, err
	}

	// Started. Runs the idempotent operation and save the result.
	if ok {
		v, err = r.handler.Query(ctx, req)
		if err != nil {
			// Delete the lock on fail.
			return v, errors.Join(err, r.store.Unlock(ctx, key))
		}

		// Save the request/response pair on success.
		return v, r.save(ctx, key, req, v, r.retention)
	}

	// The lock is acquired. The request may be in flight, or already completed.
	return r.load(ctx, key, req)
}

func (r *Query[T, U]) save(ctx context.Context, key string, req T, res U, timeout time.Duration) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	d := data[U]{
		Status:   Success,
		Request:  hash(b),
		Response: res,
	}

	return r.store.Save(ctx, key, d, timeout)
}

func (r *Query[T, U]) load(ctx context.Context, key string, req T) (U, error) {
	var v U
	d, err := r.store.Load(ctx, key)
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
