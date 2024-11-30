package idempotent

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Store interface {
	Do(ctx context.Context, key string, fn func(context.Context, []byte) ([]byte, error), req []byte) (res []byte, loaded bool, err error)
}

type Handler[T, V any] struct {
	s  Store
	fn func(ctx context.Context, req T) (V, error)
}

func NewHandler[T, V any](client *redis.Client, fn func(ctx context.Context, req T) (V, error), keepTTL, lockTTL time.Duration) *Handler[T, V] {
	store := NewRedisStore(client)
	store.KeepTTL = keepTTL
	store.LockTTL = lockTTL

	return &Handler[T, V]{
		s:  store,
		fn: fn,
	}
}

func (h *Handler[T, V]) Handle(ctx context.Context, key string, req T) (res V, shared bool, err error) {
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
	}, reqBytes)
	if err != nil {
		return
	}

	err = json.Unmarshal(resBytes, &res)
	return
}
