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

type Unique struct {
	client *redis.Client
}

func NewUnique(client *redis.Client) *Unique {
	return &Unique{
		client: client,
	}
}

func (u *Unique) Store(ctx context.Context, key, val string) (bool, error) {
	n, err := u.client.PFAdd(ctx, key, val).Result()
	return n == 1, err
}

func (u *Unique) Load(ctx context.Context, key string) (card int64, err error) {
	return u.client.PFCount(ctx, key).Result()
}
