package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// HyperLogLog is used to track unique occurences (e.g. page views).
// If we want to track the non-unique occurences (e.g. number of API calls),
// use count-min-sketch instead.
type HyperLogLog struct {
	Client *redis.Client
}

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
