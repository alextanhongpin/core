package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var _ store[any] = (*redisStore[any])(nil)

type redisStore[T any] struct {
	client *redis.Client
	prefix string
}

func NewRedisStore[T any](client *redis.Client) store[T] {
	return &redisStore[T]{
		client: client,
	}
}

func (s *redisStore[T]) Lock(ctx context.Context, key string, lockTimeout time.Duration) (bool, error) {
	ok, err := s.client.SetNX(ctx, key, fmt.Sprintf(`{"status":%q}`, Started), lockTimeout).Result()
	return ok, err
}

func (s *redisStore[T]) Unlock(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *redisStore[T]) Load(ctx context.Context, key string) (*data[T], error) {
	b, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var d data[T]
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}

	return &d, nil
}

func (s *redisStore[T]) Save(ctx context.Context, key string, d data[T], duration time.Duration) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, string(b), duration).Err()
}
