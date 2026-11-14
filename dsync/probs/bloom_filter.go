package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// BloomFilter answers "have I seen this item before?" using little memory
// (Redis BF.* commands). A "no" is always correct; a "yes" can be wrong with
// a small probability (a false positive). Items cannot be removed; use
// CuckooFilter if you need deletion, or HyperLogLog if you only need to
// count distinct items.
//
// Add creates a missing filter with the Redis defaults (1% error rate,
// capacity 100, growing as needed). Call Reserve first to size it for your
// data. A BloomFilter is safe for concurrent use.
type BloomFilter struct {
	client *redis.Client
}

// NewBloomFilter returns a BloomFilter that uses client. The client is
// borrowed: the caller keeps ownership and must close it. It returns an
// error if client is nil.
func NewBloomFilter(client *redis.Client) (*BloomFilter, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &BloomFilter{client: client}, nil
}

// MustNewBloomFilter is like NewBloomFilter but panics on error.
func MustNewBloomFilter(client *redis.Client) *BloomFilter {
	v, err := NewBloomFilter(client)
	if err != nil {
		panic(err)
	}
	return v
}

// Add adds value to the filter at key (BF.ADD). It returns true if value
// was not in the filter before, false if it was (or is a false positive).
func (bf *BloomFilter) Add(ctx context.Context, key string, value any) (bool, error) {
	return bf.client.BFAdd(ctx, key, value).Result()
}

// MAdd is like Add for several values at once (BF.MADD). The results are
// in the same order as values.
func (bf *BloomFilter) MAdd(ctx context.Context, key string, values ...any) ([]bool, error) {
	return bf.client.BFMAdd(ctx, key, values...).Result()
}

// Exists reports whether value may be in the filter (BF.EXISTS). false
// means definitely not added; true means probably added. A missing key
// reports false.
func (bf *BloomFilter) Exists(ctx context.Context, key string, value any) (bool, error) {
	return bf.client.BFExists(ctx, key, value).Result()
}

// MExists is like Exists for several values at once (BF.MEXISTS). The
// results are in the same order as values.
func (bf *BloomFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error) {
	return bf.client.BFMExists(ctx, key, values...).Result()
}

// Reserve creates an empty filter at key sized for capacity items with the
// given false-positive rate (0.01 = 1%) (BF.RESERVE). Unlike the Init and
// Create methods of the other types it is not idempotent: Redis returns an
// "item exists" error if key already exists.
func (bf *BloomFilter) Reserve(ctx context.Context, key string, errorRate float64, capacity int64) (string, error) {
	return bf.client.BFReserve(ctx, key, errorRate, capacity).Result()
}
