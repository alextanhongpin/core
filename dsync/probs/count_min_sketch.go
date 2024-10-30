package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Use this to count the stream of events. Prefer this over counter (?).
// Does not track uniqueness.
type CountMinSketch struct {
	Client *redis.Client
	group  singleflight.Group
}

func NewCountMinSketch(client *redis.Client) *CountMinSketch {
	return &CountMinSketch{
		Client: client,
	}
}

func (cms *CountMinSketch) Init(ctx context.Context, key string) (string, error) {
	errorRate := 0.001
	errorProb := 0.002
	return cms.InitByProb(ctx, key, errorRate, errorProb)
}

// needs to be created?
func (cms *CountMinSketch) InitByProb(ctx context.Context, key string, errorRate, errorProbability float64) (string, error) {
	// E.g.
	// error rate of 0.1%, errorRate = 0.001
	// probability of 99.8%, error probability of 0.02%, errorProbability = 0.002
	status, err := cms.Client.CMSInitByProb(ctx, key, errorRate, errorProbability).Result()
	if CountMinSketchKeyAlreadyExistsError(err) {
		return "OK", nil
	}

	return status, err
}

func (cms *CountMinSketch) InitByDim(ctx context.Context, key string, width, depth int64) (string, error) {
	status, err := cms.Client.CMSInitByDim(ctx, key, width, depth).Result()
	if CountMinSketchKeyAlreadyExistsError(err) {
		return "OK", nil
	}

	return status, err
}

func (cms *CountMinSketch) IncrBy(ctx context.Context, key string, kvs map[any]int) ([]int64, error) {
	args := make([]any, 0, len(kvs)*2)
	for k, v := range kvs {
		args = append(args, k, v)
	}

	counts, err := cms.Client.CMSIncrBy(ctx, key, args...).Result()
	if err == nil {
		return counts, nil
	}

	if create := CountMinSketchKeyDoesNotExistError(err); !create {
		return nil, err
	}

	_, err, shared := cms.group.Do(key, func() (any, error) {
		return cms.Init(ctx, key)
	})
	if err != nil {
		return nil, err
	}

	if !shared {
		// Clear key after created.
		cms.group.Forget(key)
	}

	return cms.Client.CMSIncrBy(ctx, key, args...).Result()
}

func CountMinSketchKeyAlreadyExistsError(err error) bool {
	return redis.HasErrorPrefix(err, "CMS: key already exists")
}

func CountMinSketchKeyDoesNotExistError(err error) bool {
	return redis.HasErrorPrefix(err, "CMS: key does not exist")
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
