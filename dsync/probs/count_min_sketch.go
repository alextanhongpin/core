package probs

import (
	"context"
	"slices"

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

func (cms *CountMinSketch) Init(ctx context.Context, key string) (status string, exists bool, err error) {
	errorRate := 0.001
	errorProb := 0.002
	return cms.InitByProb(ctx, key, errorRate, errorProb)
}

// needs to be created?
func (cms *CountMinSketch) InitByProb(ctx context.Context, key string, errorRate, errorProbability float64) (status string, exists bool, err error) {
	// E.g.
	// error rate of 0.1%, errorRate = 0.001
	// probability of 99.8%, error probability of 0.02%, errorProbability = 0.002
	status, err = cms.Client.CMSInitByProb(ctx, key, errorRate, errorProbability).Result()
	if KeyAlreadyExistsError(err) {
		return OK, true, nil
	}

	return status, false, err
}

func (cms *CountMinSketch) InitByDim(ctx context.Context, key string, width, depth int64) (status string, exists bool, err error) {
	status, err = cms.Client.CMSInitByDim(ctx, key, width, depth).Result()
	if KeyAlreadyExistsError(err) {
		return OK, true, nil
	}

	return status, false, err
}

func (cms *CountMinSketch) IncrBy(ctx context.Context, key string, kvs map[string]int64) ([]int64, bool, error) {
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

	counts, err := cms.Client.CMSIncrBy(ctx, key, args...).Result()
	if err == nil {
		return counts, false, nil
	}

	created, err := cms.create(ctx, key, err)
	if err != nil {
		return nil, false, err
	}

	counts, err = cms.Client.CMSIncrBy(ctx, key, args...).Result()
	return counts, created, err
}

func (cms *CountMinSketch) Merge(ctx context.Context, destKey string, sourceKeys ...string) (string, error) {
	status, err := cms.Client.CMSMerge(ctx, destKey, sourceKeys...).Result()
	if err == nil {
		return status, nil
	}

	if _, err := cms.create(ctx, destKey, err); err != nil {
		return "", err
	}

	return cms.Merge(ctx, destKey, sourceKeys...)
}

func (cms *CountMinSketch) MergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) (string, error) {
	status, err := cms.Client.CMSMergeWithWeight(ctx, destKey, sourceKeys).Result()
	if err == nil {
		return status, nil
	}
	if _, err := cms.create(ctx, destKey, err); err != nil {
		return "", err
	}

	return cms.MergeWithWeight(ctx, destKey, sourceKeys)
}

func (cms *CountMinSketch) Query(ctx context.Context, key string, values ...any) ([]int64, error) {
	return cms.Client.CMSQuery(ctx, key, values...).Result()
}

func (cms *CountMinSketch) create(ctx context.Context, key string, err error) (bool, error) {
	if create := KeyDoesNotExistError(err); !create {
		return false, err
	}

	resp, err, _ := cms.group.Do(key, func() (any, error) {
		_, exists, err := cms.Init(ctx, key)
		return exists, err
	})
	if err != nil {
		return false, err
	}
	exists, ok := resp.(bool)
	created := ok && !exists

	return created, nil
}
