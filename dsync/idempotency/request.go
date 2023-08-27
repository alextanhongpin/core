package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/alextanhongpin/core/internal"
)

type RequestHandler[T any] internal.RequestHandlerFunc[T]

func (r RequestHandler[T]) Exec(ctx context.Context, v T) error {
	return r(ctx, v)
}

type RequestOption[T any] struct {
	LockTimeout     time.Duration
	RetentionPeriod time.Duration
	Handler         internal.RequestHandler[T]
}

type Request[T any] struct {
	store     store[any]
	lock      time.Duration
	retention time.Duration
	handler   internal.RequestHandler[T]
}

func NewRequest[T any](store store[any], opt RequestOption[T]) *Request[T] {
	if opt.LockTimeout <= 0 {
		opt.LockTimeout = 1 * time.Minute
	}

	if opt.RetentionPeriod <= 0 {
		opt.RetentionPeriod = 24 * time.Hour
	}

	return &Request[T]{
		store:     store,
		lock:      opt.LockTimeout,
		retention: opt.RetentionPeriod,
		handler:   opt.Handler,
	}
}

func (r *Request[T]) Exec(ctx context.Context, key string, req T) error {
	ok, err := r.store.Lock(ctx, key, r.lock)
	if err != nil {
		return err
	}

	if ok {
		if err = r.handler.Exec(ctx, req); err != nil {
			return errors.Join(err, r.store.Unlock(ctx, key))
		}

		return r.save(ctx, key, req, r.retention)
	}

	return r.load(ctx, key, req)
}

func (r *Request[T]) save(ctx context.Context, key string, req T, timeout time.Duration) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	d := data[any]{
		Status:  Success,
		Request: hash(b),
	}

	return r.store.Save(ctx, key, d, timeout)
}

func (r *Request[T]) load(ctx context.Context, key string, req T) error {
	d, err := r.store.Load(ctx, key)
	if err != nil {
		return err
	}

	if d.Status == Started {
		return ErrRequestInFlight
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if d.Request != hash(b) {
		return ErrRequestMismatch
	}

	return nil
}
