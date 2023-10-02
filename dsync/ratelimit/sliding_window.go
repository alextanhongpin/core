package ratelimit

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type SlidingWindow struct {
	client *redis.Client
	limit  int64
	period int64
}

type SlidingWindowOption struct {
	Limit  int64
	Period time.Duration
}

func NewSlidingWindow(client *redis.Client, opt *SlidingWindowOption) *SlidingWindow {
	sw := &SlidingWindow{
		client: client,
		limit:  opt.Limit,
		period: opt.Period.Milliseconds(),
	}

	registerFunction(client)

	return sw
}

func (s *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	keys := []string{key}
	args := []any{s.limit, s.period}
	res, err := s.client.FCall(ctx, "sliding_window", keys, args...).Int64()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}
