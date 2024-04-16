package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}

var _ store = (*RedisStore)(nil)

func (r *RedisStore) Get(ctx context.Context, key string) (*State, error) {
	m, err := r.client.HGetAll(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Null object.
		return &State{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return &State{}, nil
	}

	status, err := strconv.Atoi(m["status"])
	if err != nil {
		return nil, err
	}

	count, err := strconv.Atoi(m["count"])
	if err != nil {
		return nil, err
	}

	resetAt, err := strconv.ParseInt(m["resetAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	closeAt, err := strconv.ParseInt(m["closeAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	return &State{
		Status:  Status(status),
		Count:   count,
		ResetAt: time.UnixMilli(resetAt),
		CloseAt: time.UnixMilli(closeAt),
	}, nil
}

func (r *RedisStore) Set(ctx context.Context, key string, res *State) error {
	hSetErr := r.client.HSet(ctx, key, map[string]interface{}{
		"status":  strconv.Itoa(int(res.Status)),
		"count":   strconv.Itoa(res.Count),
		"resetAt": fmt.Sprint(res.ResetAt.UnixMilli()),
		"closeAt": fmt.Sprint(res.CloseAt.UnixMilli()),
	}).Err()
	expireErr := r.client.PExpire(ctx, key, r.ttl).Err()
	return errors.Join(hSetErr, expireErr)
}
