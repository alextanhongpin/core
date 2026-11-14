package probs

import (
	"context"
	"maps"
	"slices"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Default accuracy used by CountMinSketch.Init and by automatic creation in
// CountMinSketch.IncrBy.
const (
	// DefaultCMSErrorRate is the over-count, as a fraction of the total of
	// all increments, that an estimate may exceed (0.1%).
	DefaultCMSErrorRate = 0.001
	// DefaultCMSErrorProbability is the probability that an estimate exceeds
	// DefaultCMSErrorRate (0.2%).
	DefaultCMSErrorProbability = 0.002
)

// CountMinSketch estimates how many times each item occurred (frequency),
// for example API calls per user, using a fixed amount of memory no matter
// how many distinct items there are. It is backed by the Redis CMS.*
// commands.
//
// Estimates are never lower than the true count, but may be higher. A
// Count-Min Sketch does not tell you how many distinct items there are; use
// HyperLogLog for that.
//
// IncrBy creates a missing sketch with the default accuracy, so calling Init
// first is optional. A CountMinSketch is safe for concurrent use.
type CountMinSketch struct {
	client *redis.Client
	group  singleflight.Group
}

// NewCountMinSketch returns a CountMinSketch that uses client. The client is
// borrowed: the caller keeps ownership and must close it. It returns an
// error if client is nil.
func NewCountMinSketch(client *redis.Client) (*CountMinSketch, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &CountMinSketch{client: client}, nil
}

// MustNewCountMinSketch is like NewCountMinSketch but panics on error.
func MustNewCountMinSketch(client *redis.Client) *CountMinSketch {
	v, err := NewCountMinSketch(client)
	if err != nil {
		panic(err)
	}
	return v
}

// Init creates the sketch at key with DefaultCMSErrorRate and
// DefaultCMSErrorProbability. It is idempotent: if key already exists it
// returns OK with exists set to true and leaves the sketch unchanged.
func (cms *CountMinSketch) Init(ctx context.Context, key string) (status string, exists bool, err error) {
	return cms.InitByProb(ctx, key, DefaultCMSErrorRate, DefaultCMSErrorProbability)
}

// InitByProb creates the sketch at key sized for the given accuracy
// (CMS.INITBYPROB). errorRate is the allowed over-count as a fraction of the
// total of all increments (0.001 = 0.1%); errorProbability is the chance
// that an estimate is worse than that (0.002 = 0.2%). It is idempotent like
// Init.
func (cms *CountMinSketch) InitByProb(ctx context.Context, key string, errorRate, errorProbability float64) (status string, exists bool, err error) {
	status, err = cms.client.CMSInitByProb(ctx, key, errorRate, errorProbability).Result()
	if KeyAlreadyExistsError(err) {
		return OK, true, nil
	}

	return status, false, err
}

// InitByDim creates the sketch at key with an explicit width (counters per
// row) and depth (number of rows) (CMS.INITBYDIM). It is idempotent like
// Init.
func (cms *CountMinSketch) InitByDim(ctx context.Context, key string, width, depth int64) (status string, exists bool, err error) {
	status, err = cms.client.CMSInitByDim(ctx, key, width, depth).Result()
	if KeyAlreadyExistsError(err) {
		return OK, true, nil
	}

	return status, false, err
}

// IncrBy adds each map value to the count of its item (CMS.INCRBY) and
// returns the new estimated counts. Because a map has no order, counts are
// returned in ascending order of the item names (see sort.Strings).
//
// If the sketch does not exist it is created with Init's defaults and
// created is true.
func (cms *CountMinSketch) IncrBy(ctx context.Context, key string, kvs map[string]int64) (counts []int64, created bool, err error) {
	args := sortedPairs(kvs)

	counts, err = cms.client.CMSIncrBy(ctx, key, args...).Result()
	if err == nil {
		return counts, false, nil
	}

	if !KeyDoesNotExistError(err) {
		return nil, false, err
	}
	created, err = cms.create(key, func() (bool, error) {
		_, exists, err := cms.Init(ctx, key)
		return exists, err
	})
	if err != nil {
		return nil, false, err
	}

	counts, err = cms.client.CMSIncrBy(ctx, key, args...).Result()
	return counts, created, err
}

// Merge sums the source sketches into destKey (CMS.MERGE), replacing the
// content of destKey. All sketches must have the same width and depth. If
// destKey does not exist it is created with the dimensions of the first
// source. A missing source returns an error and leaves destKey untouched.
func (cms *CountMinSketch) Merge(ctx context.Context, destKey string, sourceKeys ...string) (string, error) {
	status, err := cms.client.CMSMerge(ctx, destKey, sourceKeys...).Result()
	if err == nil || len(sourceKeys) == 0 {
		return status, err
	}

	if err := cms.createLike(ctx, destKey, sourceKeys[0], err); err != nil {
		return "", err
	}

	return cms.client.CMSMerge(ctx, destKey, sourceKeys...).Result()
}

// MergeWithWeight is like Merge, but multiplies each source sketch (map
// key) by its weight (map value) before summing.
func (cms *CountMinSketch) MergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) (string, error) {
	status, err := cms.client.CMSMergeWithWeight(ctx, destKey, sourceKeys).Result()
	if err == nil || len(sourceKeys) == 0 {
		return status, err
	}

	first := slices.Min(slices.Collect(maps.Keys(sourceKeys)))
	if err := cms.createLike(ctx, destKey, first, err); err != nil {
		return "", err
	}

	return cms.client.CMSMergeWithWeight(ctx, destKey, sourceKeys).Result()
}

// Query returns the estimated count of each value, in the same order
// (CMS.QUERY). Values never added count as 0. It returns an error if the
// sketch does not exist.
func (cms *CountMinSketch) Query(ctx context.Context, key string, values ...any) ([]int64, error) {
	return cms.client.CMSQuery(ctx, key, values...).Result()
}

// createLike creates destKey with the same dimensions as srcKey after a
// merge failed with err because a key was missing. It returns err unchanged
// for any other failure, and the CMS.INFO error if srcKey is the missing key.
func (cms *CountMinSketch) createLike(ctx context.Context, destKey, srcKey string, err error) error {
	if !KeyDoesNotExistError(err) {
		return err
	}
	info, err := cms.client.CMSInfo(ctx, srcKey).Result()
	if err != nil {
		return err
	}
	_, err = cms.create(destKey, func() (bool, error) {
		_, exists, err := cms.InitByDim(ctx, destKey, info.Width, info.Depth)
		return exists, err
	})
	return err
}

// create runs init once per key across concurrent callers and reports
// whether the sketch was created (rather than found to already exist).
func (cms *CountMinSketch) create(key string, init func() (exists bool, err error)) (bool, error) {
	resp, err, _ := cms.group.Do(key, func() (any, error) {
		return init()
	})
	if err != nil {
		return false, err
	}
	exists, ok := resp.(bool)
	return ok && !exists, nil
}

// sortedPairs flattens kvs into item, increment pairs ordered by item, so
// the command and its reply have a deterministic order.
func sortedPairs(kvs map[string]int64) []any {
	keys := slices.Sorted(maps.Keys(kvs))
	args := make([]any, 0, len(kvs)*2)
	for _, k := range keys {
		args = append(args, k, kvs[k])
	}
	return args
}
