package probs

import (
	"context"
	"math"
	"slices"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// https://www.moesif.com/blog/api-product-management/api-analytics/Using-Time-Series-Charts-to-Explore-API-Usage/
// app:data_type:key:value:key:value:time:range
// app:hll:pageviews:2024-05-01 page

type TopK struct {
	Client *redis.Client
	group  singleflight.Group
}

func NewTopK(client *redis.Client) *TopK {
	return &TopK{
		Client: client,
	}
}

func (t *TopK) Add(ctx context.Context, key string, values ...any) ([]string, error) {
	vals, err := t.Client.TopKAdd(ctx, key, values...).Result()
	if err == nil {
		return vals, nil
	}

	if err := t.create(ctx, key, err); err != nil {
		return nil, err
	}

	return t.Client.TopKAdd(ctx, key, values...).Result()
}

func (t *TopK) Count(ctx context.Context, key string, values ...any) ([]int64, error) {
	return t.Client.TopKCount(ctx, key, values...).Result()
}

func (t *TopK) IncrBy(ctx context.Context, key string, kvs map[string]int64) ([]string, error) {
	keys := make([]string, 0, len(kvs))
	for k := range kvs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	args := make([]any, len(kvs)*2)
	for i, k := range keys {
		args[i*2] = k
		args[i*2+1] = kvs[k]
	}

	vals, err := t.Client.TopKIncrBy(ctx, key, args...).Result()
	if err == nil {
		return vals, nil
	}
	if err := t.create(ctx, key, err); err != nil {
		return nil, err
	}

	return t.Client.TopKIncrBy(ctx, key, args...).Result()
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

func (t *TopK) Create(ctx context.Context, key string, k int64) (string, error) {
	logK := math.Log(float64(k))
	width := int64(float64(k) * logK)
	depth := int64(max(logK, 5))
	decay := 0.9
	status, err := t.Client.TopKReserveWithOptions(ctx, key, k, width, depth, decay).Result()
	if KeyAlreadyExistsError(err) {
		return OK, nil
	}

	return status, err
}

func (t *TopK) create(ctx context.Context, key string, err error) error {
	if create := KeyDoesNotExistError(err); !create {
		return err
	}

	_, err, shared := t.group.Do(key, func() (any, error) {
		return t.Create(ctx, key, 10)
	})
	if err != nil {
		return err
	}

	if !shared {
		// Clear key after created.
		t.group.Forget(key)
	}

	return nil
}
