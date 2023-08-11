package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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

var keyTemplate = Key("i9y:%s")

type data[T any] struct {
	Status   Status `json:"status"`
	Request  string `json:"request,omitempty"`
	Response T      `json:"response,omitempty"`
}

type store[T any] interface {
	lock(ctx context.Context, idempotencyKey string, lockTimeout time.Duration) (bool, error)
	unlock(ctx context.Context, idempotencyKey string) error
	load(ctx context.Context, idempotencyKey string) (*data[T], error)
	save(ctx context.Context, idempotencyKey string, d data[T], duration time.Duration) error
}

type redisStore[T any] struct {
	client *redis.Client
}

func newRedisStore[T any](client *redis.Client) *redisStore[T] {
	return &redisStore[T]{
		client: client,
	}
}

func (s *redisStore[T]) lock(ctx context.Context, idempotencyKey string, lockTimeout time.Duration) (bool, error) {
	key := keyTemplate.Format(idempotencyKey)

	ok, err := s.client.SetNX(ctx, key, fmt.Sprintf(`{"status":%q}`, Started), lockTimeout).Result()
	return ok, err
}

func (s *redisStore[T]) unlock(ctx context.Context, idempotencyKey string) error {
	key := keyTemplate.Format(idempotencyKey)

	return s.client.Del(ctx, key).Err()
}

func (s *redisStore[T]) load(ctx context.Context, idempotencyKey string) (*data[T], error) {
	key := keyTemplate.Format(idempotencyKey)

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

func (s *redisStore[T]) save(ctx context.Context, idempotencyKey string, d data[T], duration time.Duration) error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	key := keyTemplate.Format(idempotencyKey)
	return s.client.Set(ctx, key, string(b), duration).Err()
}

func hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	b := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}
