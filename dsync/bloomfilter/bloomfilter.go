package bloomfilter

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

type BloomFilter struct {
	Client *redis.Client
}

func NewBloomFilter(client *redis.Client) *BloomFilter {
	return &BloomFilter{
		Client: client,
	}
}

func (b *BloomFilter) Create(ctx context.Context, key string, errorRate float64, capacity int64) error {
	// BF.RESERVE bikes:models 0.001 1000000
	return b.Client.Do(ctx, "BF.RESERVE", key, errorRate, capacity).Err()
}

func (b *BloomFilter) Add(ctx context.Context, key, val string) (inserted bool, err error) {
	return b.Client.Do(ctx, "BF.ADD", key, val).Bool()
}

func (b *BloomFilter) Exists(ctx context.Context, key, val string) (exists bool, err error) {
	return b.Client.Do(ctx, "BF.EXISTS", key, val).Bool()
}
