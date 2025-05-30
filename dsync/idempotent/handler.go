package idempotent

import (
	"cmp"
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	lockTTL = 10 * time.Second
	keepTTL = 24 * time.Hour
)

type Store interface {
	Do(ctx context.Context, key string, fn func(context.Context, []byte) ([]byte, error), req []byte, lockTTL, keepTTL time.Duration) (res []byte, loaded bool, err error)
}

type HandlerOptions struct {
	LockTTL time.Duration
	KeepTTL time.Duration
}

type Handler[T, V any] struct {
	s    Store
	fn   func(ctx context.Context, req T) (V, error)
	opts *HandlerOptions
}

func NewHandler[T, V any](client *redis.Client, fn func(ctx context.Context, req T) (V, error), opts *HandlerOptions) *Handler[T, V] {
	opts = cmp.Or(opts, &HandlerOptions{})
	opts.LockTTL = cmp.Or(opts.LockTTL, lockTTL)
	opts.KeepTTL = cmp.Or(opts.KeepTTL, keepTTL)

	return &Handler[T, V]{
		s:    NewRedisStore(client),
		fn:   fn,
		opts: opts,
	}
}

func (h *Handler[T, V]) Handle(ctx context.Context, key string, req T) (res V, shared bool, err error) {
	if key == "" {
		return res, false, ErrEmptyKey
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return res, shared, err
	}

	var resBytes []byte
	resBytes, shared, err = h.s.Do(ctx, key, func(ctx context.Context, reqBytes []byte) ([]byte, error) {
		var req T
		if err := json.Unmarshal(reqBytes, &req); err != nil {
			return nil, err
		}

		res, err := h.fn(ctx, req)
		if err != nil {
			return nil, err
		}

		return json.Marshal(res)
	}, reqBytes, h.opts.LockTTL, h.opts.KeepTTL)
	if err != nil {
		return
	}

	err = json.Unmarshal(resBytes, &res)
	return
}
