package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type RequestReplyOption[T comparable, U any] struct {
	LockTimeout     time.Duration
	RetentionPeriod time.Duration
	Handler         func(ctx context.Context, v T) (U, error)
}

type RequestReply[T comparable, U any] struct {
	store     store[U]
	lock      time.Duration
	retention time.Duration
	handler   func(ctx context.Context, v T) (U, error)
}

func NewRequestReply[T comparable, U any](store store[U], opt RequestReplyOption[T, U]) *RequestReply[T, U] {
	if opt.LockTimeout <= 0 {
		opt.LockTimeout = 1 * time.Minute
	}

	if opt.RetentionPeriod <= 0 {
		opt.RetentionPeriod = 24 * time.Hour
	}

	return &RequestReply[T, U]{
		store:     store,
		lock:      opt.LockTimeout,
		retention: opt.RetentionPeriod,
		handler:   opt.Handler,
	}
}

func (r *RequestReply[T, U]) Exec(ctx context.Context, key string, req T) (U, error) {
	// Sets the idempotency operation status to "started".
	// Can only be executed by one client.
	var v U
	ok, err := r.store.Lock(ctx, key, r.lock)
	if err != nil {
		return v, err
	}

	// Started. Runs the idempotent operation and save the result.
	if ok {
		v, err = r.handler(ctx, req)
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

func (r *RequestReply[T, U]) save(ctx context.Context, key string, req T, res U, timeout time.Duration) error {
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

func (r *RequestReply[T, U]) load(ctx context.Context, key string, req T) (U, error) {
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
