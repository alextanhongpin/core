package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Similar to bloomfilter, except it can be deleted.
type CuckooFilter struct {
	Client *redis.Client
}

func (cf *CuckooFilter) Add(ctx context.Context, key, value string) (bool, error) {
	return cf.Client.CFAdd(ctx, key, value).Result()
}

func (cf *CuckooFilter) AddNX(ctx context.Context, key, value string) (bool, error) {
	return cf.Client.CFAddNX(ctx, key, value).Result()
}

func (cf *CuckooFilter) Exists(ctx context.Context, key, value string) (bool, error) {
	return cf.Client.CFExists(ctx, key, value).Result()
}

func (cf *CuckooFilter) Count(ctx context.Context, key, value string) (int64, error) {
	return cf.Client.CFCount(ctx, key, value).Result()
}

func (cf *CuckooFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error) {
	return cf.Client.CFMExists(ctx, key, values...).Result()
}

func (cf *CuckooFilter) Delete(ctx context.Context, key, value string) (bool, error) {
	return cf.Client.CFDel(ctx, key, value).Result()
}
