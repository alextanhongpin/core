package probs

import (
	"context"
	"math"

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

func (t *TopK) IncrBy(ctx context.Context, key string, evt *Event) ([]string, error) {
	return t.Client.TopKIncrBy(ctx, key, evt.Data()...).Result()
}

func (t *TopK) List(ctx context.Context, key string) ([]string, error) {
	return t.Client.TopKList(ctx, key).Result()
}

func (t *TopK) ListWithCount(ctx context.Context, key string) (map[string]int64, error) {
	return t.Client.TopKListWithCount(ctx, key).Result()
}

// Query returns if the values exists in the top list.
func (t *TopK) Query(ctx context.Context, key string, values ...any) ([]bool, error) {
	return t.Client.TopKQuery(ctx, key, values...).Result()
}

func (t *TopK) Reserve(ctx context.Context, key string, k int64) (string, error) {
	return t.Client.TopKReserve(ctx, key, k).Result()
}

func (t *TopK) ReserveWithOptions(ctx context.Context, key string, k, width, depth int64, decay float64) (string, error) {
	return t.Client.TopKReserveWithOptions(ctx, key, k, width, depth, decay).Result()
}

// needs to be created?
func (t *TopK) Create(ctx context.Context, key string, k int64) (string, error) {
	logK := math.Log(float64(k))
	width := int64(float64(k) * logK)
	depth := int64(max(logK, 5))
	decay := 0.9
	return t.Client.TopKReserveWithOptions(ctx, key, k, width, depth, decay).Result()
}
