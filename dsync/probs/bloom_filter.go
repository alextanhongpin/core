package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Use this to track unique page views/actions. For unique count, use hyperloglog
type BloomFilter struct {
	Client *redis.Client
}

func (bf *BloomFilter) Add(ctx context.Context, key string, value any) (bool, error) {
	return bf.Client.BFAdd(ctx, key, value).Result()
}

func (bf *BloomFilter) MAdd(ctx context.Context, key string, values ...any) ([]bool, error) {
	return bf.Client.BFMAdd(ctx, key, values...).Result()
}

func (bf *BloomFilter) Exists(ctx context.Context, key string, value any) (bool, error) {
	return bf.Client.BFExists(ctx, key, value).Result()
}

func (bf *BloomFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error) {
	return bf.Client.BFMExists(ctx, key, values...).Result()
}

func (bf *BloomFilter) Reserve(ctx context.Context, key string, errorRate float64, capacity int64) (string, error) {
	return bf.Client.BFReserve(ctx, key, errorRate, capacity).Result()
}
