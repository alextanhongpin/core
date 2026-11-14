package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// CuckooFilter is like BloomFilter, but items can also be deleted and
// counted (Redis CF.* commands). Membership answers can be false positives,
// never false negatives (unless you delete an item that was never added).
//
// Add creates a missing filter with the Redis default capacity (1024,
// growing as needed). Call Reserve first to size it for your data. A
// CuckooFilter is safe for concurrent use.
type CuckooFilter struct {
	client *redis.Client
}

// NewCuckooFilter returns a CuckooFilter that uses client. The client is
// borrowed: the caller keeps ownership and must close it. It returns an
// error if client is nil.
func NewCuckooFilter(client *redis.Client) (*CuckooFilter, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &CuckooFilter{client: client}, nil
}

// MustNewCuckooFilter is like NewCuckooFilter but panics on error.
func MustNewCuckooFilter(client *redis.Client) *CuckooFilter {
	v, err := NewCuckooFilter(client)
	if err != nil {
		panic(err)
	}
	return v
}

// Add adds value to the filter at key (CF.ADD), even if it is already
// there: adding twice stores two copies, which Count reports and which take
// two Deletes to remove. Use AddNX to add at most once.
func (cf *CuckooFilter) Add(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFAdd(ctx, key, value).Result()
}

// AddNX adds value only if it is not already in the filter (CF.ADDNX). It
// returns true if value was added, false if it (probably) already existed.
func (cf *CuckooFilter) AddNX(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFAddNX(ctx, key, value).Result()
}

// Exists reports whether value may be in the filter (CF.EXISTS). false
// means definitely not present. A missing key reports false.
func (cf *CuckooFilter) Exists(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFExists(ctx, key, value).Result()
}

// Count returns how many copies of value the filter holds (CF.COUNT). It
// may over-count, but never under-counts.
func (cf *CuckooFilter) Count(ctx context.Context, key, value string) (int64, error) {
	return cf.client.CFCount(ctx, key, value).Result()
}

// MExists is like Exists for several values at once (CF.MEXISTS). The
// results are in the same order as values.
func (cf *CuckooFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error) {
	return cf.client.CFMExists(ctx, key, values...).Result()
}

// Delete removes one copy of value from the filter (CF.DEL). It returns
// false if value was not found. Only delete items you know were added;
// deleting a false positive removes another item's entry.
func (cf *CuckooFilter) Delete(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFDel(ctx, key, value).Result()
}

// Reserve creates an empty filter at key with room for about capacity
// items (CF.RESERVE). Redis returns an "item exists" error if key already
// exists.
func (cf *CuckooFilter) Reserve(ctx context.Context, key string, capacity int64) (string, error) {
	return cf.client.CFReserve(ctx, key, capacity).Result()
}
