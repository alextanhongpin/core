package ab

import (
	"context"

	redis "github.com/redis/go-redis/v9"
	"github.com/spaolacci/murmur3"
)

func Hash(key string, size uint64) uint64 {
	return murmur3.Sum64([]byte(key)) % size
}

func Rollout(key string, percentage uint64) bool {
	return percentage > 0 && Hash(key, 100) <= percentage
}

type Counter struct {
	client *redis.Client
}

func NewCounter(client *redis.Client) *Counter {
	return &Counter{
		client: client,
	}
}

func (c *Counter) Store(ctx context.Context, key, val string) (stored bool, err error) {
	n, err := c.client.PFAdd(ctx, key, val).Result()
	return n == 1, err
}

func (c *Counter) Load(ctx context.Context, key string) (count int64, err error) {
	return c.client.PFCount(ctx, key).Result()
}
