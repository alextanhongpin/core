package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type RequestOption[T any] struct {
	Lock    time.Duration
	TTL     time.Duration
	Handler func(ctx context.Context, v T) error
}

type Request[T any] struct {
	store   store[any]
	lock    time.Duration
	ttl     time.Duration
	handler func(ctx context.Context, v T) error
}

func NewRequest[T any](store store[any], opt RequestOption[T]) *Request[T] {
	if opt.Lock <= 0 {
		opt.Lock = 1 * time.Minute
	}

	if opt.TTL <= 0 {
		opt.TTL = 24 * time.Hour
	}

	return &Request[T]{
		store:   store,
		lock:    opt.Lock,
		ttl:     opt.TTL,
		handler: opt.Handler,
	}
}

func (r *Request[T]) Exec(ctx context.Context, key string, req T) error {
	ok, err := r.store.Lock(ctx, key, r.lock)
	if err != nil {
		return err
	}

	if ok {
		if err = r.handler(ctx, req); err != nil {
			return errors.Join(err, r.store.Unlock(ctx, key))
		}

		return r.save(ctx, key, req, r.ttl)
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
