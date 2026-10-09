package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Similar to bloomfilter, except it can be deleted.
type CuckooFilter struct {
	client *redis.Client
}

// NewCuckooFilter borrows client; the caller owns its lifetime.
func NewCuckooFilter(client *redis.Client) (*CuckooFilter, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &CuckooFilter{client: client}, nil
}
func MustNewCuckooFilter(client *redis.Client) *CuckooFilter {
	v, err := NewCuckooFilter(client)
	if err != nil {
		panic(err)
	}
	return v
}

func (cf *CuckooFilter) Add(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFAdd(ctx, key, value).Result()
}

func (cf *CuckooFilter) AddNX(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFAddNX(ctx, key, value).Result()
}

func (cf *CuckooFilter) Exists(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFExists(ctx, key, value).Result()
}

func (cf *CuckooFilter) Count(ctx context.Context, key, value string) (int64, error) {
	return cf.client.CFCount(ctx, key, value).Result()
}

func (cf *CuckooFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error) {
	return cf.client.CFMExists(ctx, key, values...).Result()
}

func (cf *CuckooFilter) Delete(ctx context.Context, key, value string) (bool, error) {
	return cf.client.CFDel(ctx, key, value).Result()
}
