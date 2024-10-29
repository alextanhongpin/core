package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

type HyperLogLog struct {
	Client *redis.Client
}

// use this to track unique page views.
func NewHyperLogLog(client *redis.Client) *HyperLogLog {
	return &HyperLogLog{
		Client: client,
	}
}

func (c *HyperLogLog) Add(ctx context.Context, key string, values ...any) (int64, error) {
	return c.Client.PFAdd(ctx, key, values).Result()
}

func (c *HyperLogLog) Count(ctx context.Context, keys ...string) (int64, error) {
	return c.Client.PFCount(ctx, keys...).Result()
}

func (c *HyperLogLog) Merge(ctx context.Context, destKey string, srcKeys ...string) (string, error) {
	return c.Client.PFMerge(ctx, destKey, srcKeys...).Result()
}
