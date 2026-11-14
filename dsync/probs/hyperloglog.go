package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// HyperLogLog estimates how many distinct items were added (cardinality),
// for example unique visitors per day, using at most 12 KB per key and a
// standard error of about 0.81% (Redis PF* commands). It cannot tell you
// whether a specific item was added; use BloomFilter for that. To count
// repeated occurrences of each item, use CountMinSketch.
//
// It uses core Redis commands, so no module is needed. A HyperLogLog is
// safe for concurrent use.
type HyperLogLog struct {
	client *redis.Client
}

// NewHyperLogLog returns a HyperLogLog that uses client. The client is
// borrowed: the caller keeps ownership and must close it. It returns an
// error if client is nil.
func NewHyperLogLog(client *redis.Client) (*HyperLogLog, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &HyperLogLog{client: client}, nil
}

// MustNewHyperLogLog is like NewHyperLogLog but panics on error.
func MustNewHyperLogLog(client *redis.Client) *HyperLogLog {
	v, err := NewHyperLogLog(client)
	if err != nil {
		panic(err)
	}
	return v
}

// Add adds values to the HyperLogLog at key, creating it if needed (PFADD).
// It returns 1 if the estimated count changed, otherwise 0.
func (c *HyperLogLog) Add(ctx context.Context, key string, values ...any) (int64, error) {
	return c.client.PFAdd(ctx, key, values...).Result()
}

// Count returns the estimated number of distinct items (PFCOUNT). With
// several keys it estimates the size of their union. A missing key counts
// as 0.
func (c *HyperLogLog) Count(ctx context.Context, keys ...string) (int64, error) {
	return c.client.PFCount(ctx, keys...).Result()
}

// Merge stores the union of srcKeys (and destKey's existing content) in
// destKey (PFMERGE), for example to roll daily visitors up into a weekly
// count.
func (c *HyperLogLog) Merge(ctx context.Context, destKey string, srcKeys ...string) (string, error) {
	return c.client.PFMerge(ctx, destKey, srcKeys...).Result()
}
