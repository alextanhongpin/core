package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var (
	ErrRequestInFlight = errors.New("idempotency: request in flight")
	ErrRequestMismatch = errors.New("idempotency: request payload mismatch")
)

type Key string

func (k Key) Format(args ...any) string {
	return fmt.Sprintf(string(k), args...)
}

type Status string

const (
	Started Status = "started"
	Success Status = "success"
)

var keyTemplate = Key("idempotency:%s")

type data[T, V any] struct {
	Status   Status `json:"status"`
	Request  T      `json:"request,omitempty"`
	Response V      `json:"response,omitempty"`
}

type store[T, V any] interface {
	lock(ctx context.Context, idempotencyKey string, lockTimeout time.Duration) (bool, error)
	unlock(ctx context.Context, idempotencyKey string) error
	load(ctx context.Context, idempotencyKey string) (*data[T, V], error)
	save(ctx context.Context, idempotencyKey string, d data[T, V], duration time.Duration) error
}

type redisStore[T any, V any] struct {
	client *redis.Client
}

func newRedisStore[T, V any](client *redis.Client) *redisStore[T, V] {
	return &redisStore[T, V]{
		client: client,
	}
}

func (s *redisStore[T, V]) lock(ctx context.Context, idempotencyKey string, lockTimeout time.Duration) (bool, error) {
	key := keyTemplate.Format(idempotencyKey)

	ok, err := s.client.SetNX(ctx, key, fmt.Sprintf(`{"status":%q}`, Started), lockTimeout).Result()
	return ok, err
}

func (s *redisStore[T, V]) unlock(ctx context.Context, idempotencyKey string) error {
	key := keyTemplate.Format(idempotencyKey)

	return s.client.Del(ctx, key).Err()
}

func (s *redisStore[T, V]) load(ctx context.Context, idempotencyKey string) (*data[T, V], error) {

	key := keyTemplate.Format(idempotencyKey)

	b, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var d data[T, V]
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}

	return &d, nil
}

func (s *redisStore[T, V]) save(ctx context.Context, idempotencyKey string, d data[T, V], duration time.Duration) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	key := keyTemplate.Format(idempotencyKey)
	return s.client.Set(ctx, key, string(b), duration).Err()
}
