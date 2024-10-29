package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Use this to count the stream of events. Prefer this over counter (?).
// Does not track uniqueness.
type CountMinSketch struct {
	Client *redis.Client
}

func NewCountMinSketch(client *redis.Client) *CountMinSketch {
	return &CountMinSketch{
		Client: client,
	}
}

func (cms *CountMinSketch) InitByProb(ctx context.Context, key string, errorRate, probability float64) (string, error) {
	return cms.Client.CMSInitByProb(ctx, key, errorRate, probability).Result()
}

func (cms *CountMinSketch) InitByDim(ctx context.Context, key string, width, depth int64) (string, error) {
	return cms.Client.CMSInitByDim(ctx, key, width, depth).Result()
}

type Tuple[K, V any] struct {
	K K
	V V
}

func (cms *CountMinSketch) IncrBy(ctx context.Context, key string, values ...Tuple[any, int]) ([]int64, error) {
	args := make([]any, len(values)*2)
	for i, v := range values {
		args[i*2] = v.K
		args[i*2+1] = v.V
	}

	return cms.Client.CMSIncrBy(ctx, key, args...).Result()
}

func (cms *CountMinSketch) Merge(ctx context.Context, destKey string, sourceKeys ...string) (string, error) {
	return cms.Client.CMSMerge(ctx, destKey, sourceKeys...).Result()
}

func (cms *CountMinSketch) MergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) (string, error) {
	return cms.Client.CMSMergeWithWeight(ctx, destKey, sourceKeys).Result()
}

func (cms *CountMinSketch) Query(ctx context.Context, key string, values ...any) ([]int64, error) {
	return cms.Client.CMSQuery(ctx, key, values...).Result()
}
