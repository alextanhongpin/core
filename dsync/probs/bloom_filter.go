package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// BloomFilter is used to track unique entries, up to an acceptable error rate.
// Use bloom filter to check for existence. To track count of unique entries
// (e.g. page views), use hyperloglog instead.
type BloomFilter struct {
	client *redis.Client
}

// NewBloomFilter borrows client; the caller owns its lifetime.
func NewBloomFilter(client *redis.Client) (*BloomFilter, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &BloomFilter{client: client}, nil
}
func MustNewBloomFilter(client *redis.Client) *BloomFilter {
	v, err := NewBloomFilter(client)
	if err != nil {
		panic(err)
	}
	return v
}

func (bf *BloomFilter) Add(ctx context.Context, key string, value any) (bool, error) {
	return bf.client.BFAdd(ctx, key, value).Result()
}

func (bf *BloomFilter) MAdd(ctx context.Context, key string, values ...any) ([]bool, error) {
	return bf.client.BFMAdd(ctx, key, values...).Result()
}

func (bf *BloomFilter) Exists(ctx context.Context, key string, value any) (bool, error) {
	return bf.client.BFExists(ctx, key, value).Result()
}

func (bf *BloomFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error) {
	return bf.client.BFMExists(ctx, key, values...).Result()
}

func (bf *BloomFilter) Reserve(ctx context.Context, key string, errorRate float64, capacity int64) (string, error) {
	return bf.client.BFReserve(ctx, key, errorRate, capacity).Result()
}
