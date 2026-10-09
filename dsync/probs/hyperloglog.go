package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// HyperLogLog is used to track unique occurences (e.g. page views).
// If we want to track the non-unique occurences (e.g. number of API calls),
// use count-min-sketch instead.
type HyperLogLog struct {
	client *redis.Client
}

// NewHyperLogLog borrows client; the caller owns its lifetime.
func NewHyperLogLog(client *redis.Client) (*HyperLogLog, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &HyperLogLog{client: client}, nil
}
func MustNewHyperLogLog(client *redis.Client) *HyperLogLog {
	v, err := NewHyperLogLog(client)
	if err != nil {
		panic(err)
	}
	return v
}

func (c *HyperLogLog) Add(ctx context.Context, key string, values ...any) (int64, error) {
	return c.client.PFAdd(ctx, key, values...).Result()
}

func (c *HyperLogLog) Count(ctx context.Context, keys ...string) (int64, error) {
	return c.client.PFCount(ctx, keys...).Result()
}

func (c *HyperLogLog) Merge(ctx context.Context, destKey string, srcKeys ...string) (string, error) {
	return c.client.PFMerge(ctx, destKey, srcKeys...).Result()
}
