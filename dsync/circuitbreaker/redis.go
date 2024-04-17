package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrConcurrentWrite = errors.New("circuitbreaker: concurrent write")

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
		status:  Status(status),
		Status:  Status(status),
		Count:   count,
		ResetAt: time.UnixMilli(resetAt),
		CloseAt: time.UnixMilli(closeAt),
	}, nil
}

func (r *RedisStore) Set(ctx context.Context, key string, res *State) error {
	keys := []string{
		key,
		strconv.Itoa(int(res.status)),
		fmt.Sprint(formatMs(r.ttl)),
	}
	argv := []any{
		"status", strconv.Itoa(int(res.Status)),
		"count", strconv.Itoa(res.Count),
		"resetAt", fmt.Sprint(res.ResetAt.UnixMilli()),
		"closeAt", fmt.Sprint(res.CloseAt.UnixMilli()),
	}
	err := script.Run(ctx, r.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrConcurrentWrite
	}

	return err
}

// copied from redis source code
func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		return 1
	}

	return int64(dur / time.Millisecond)
}
