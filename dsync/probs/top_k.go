package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// https://www.moesif.com/blog/api-product-management/api-analytics/Using-Time-Series-Charts-to-Explore-API-Usage/
// app:data_type:key:value:key:value:time:range
// app:hll:pageviews:2024-05-01 page

type TopK struct {
	Client *redis.Client
}

func NewTopK(client *redis.Client) *TopK {
	return &TopK{
		Client: client,
	}
}

func (t *TopK) Add(ctx context.Context, key string, values ...any) ([]string, error) {
	return t.Client.TopKAdd(ctx, key, values...).Result()
}

func (t *TopK) Count(ctx context.Context, key string, values ...any) ([]int64, error) {
	return t.Client.TopKCount(ctx, key, values...).Result()
}

func (t *TopK) IncrBy(ctx context.Context, key string, values ...Tuple[any, int]) ([]string, error) {
	args := make([]any, len(values)*2)
	for i, v := range values {
		args[i*2] = v.K
		args[i*2+1] = v.V
	}

	return t.Client.TopKIncrBy(ctx, key, args...).Result()
}

func (t *TopK) List(ctx context.Context, key string) ([]string, error) {
	return t.Client.TopKList(ctx, key).Result()
}

func (t *TopK) ListWithCount(ctx context.Context, key string) (map[string]int64, error) {
	return t.Client.TopKListWithCount(ctx, key).Result()
}

func (t *TopK) Query(ctx context.Context, key string, values ...any) ([]bool, error) {
	return t.Client.TopKQuery(ctx, key, values...).Result()
}

func (t *TopK) Reserve(ctx context.Context, key string, k int64) (string, error) {
	return t.Client.TopKReserve(ctx, key, k).Result()
}

func (t *TopK) ReserveWithOptions(ctx context.Context, key string, k, width, depth int64, decay float64) (string, error) {
	return t.Client.TopKReserveWithOptions(ctx, key, k, width, depth, decay).Result()
}
