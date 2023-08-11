package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type CmdOption[T any] struct {
	LockTimeout     time.Duration
	RetentionPeriod time.Duration
	Handler         func(ctx context.Context, req T) error
}

type Cmd[T any] struct {
	store     store[any]
	lock      time.Duration
	retention time.Duration
	handler   func(ctx context.Context, req T) error
}

func NewCmd[T any](client *redis.Client, opt CmdOption[T]) *Cmd[T] {
	if opt.LockTimeout <= 0 {
		opt.LockTimeout = 1 * time.Minute
	}

	if opt.RetentionPeriod <= 0 {
		opt.RetentionPeriod = 24 * time.Hour
	}

	return &Cmd[T]{
		store:     newRedisStore[any](client),
		lock:      opt.LockTimeout,
		retention: opt.RetentionPeriod,
		handler:   opt.Handler,
	}
}

func (r *Cmd[T]) Exec(ctx context.Context, key string, req T) error {
	ok, err := r.store.lock(ctx, key, r.lock)
	if err != nil {
		return err
	}

	if ok {
		if err = r.handler(ctx, req); err != nil {
			return errors.Join(err, r.store.unlock(ctx, key))
		}

		return r.save(ctx, key, req, r.retention)
	}

	return r.load(ctx, key, req)
}

func (r *Cmd[T]) save(ctx context.Context, key string, req T, timeout time.Duration) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	d := data[any]{
		Status:  Success,
		Request: hash(b),
	}

	return r.store.save(ctx, key, d, timeout)
}

func (r *Cmd[T]) load(ctx context.Context, key string, req T) error {
	d, err := r.store.load(ctx, key)
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
