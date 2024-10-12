package cache

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type JSON struct {
	Cache Cacheable
}

func NewJSON(client *redis.Client) *JSON {
	return &JSON{
		Cache: New(client),
	}
}

func (s *JSON) Load(ctx context.Context, key string, v any) error {
	str, err := s.Cache.Load(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(str), &v)
}

func (s *JSON) Store(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Cache.Store(ctx, key, b, ttl)
}

func (s *JSON) LoadAndDelete(ctx context.Context, key string, value any) (loaded bool, err error) {
	str, loaded, err := s.Cache.LoadAndDelete(ctx, key)
	if err != nil {
		return false, err
	}
	if !loaded {
		return false, nil
	}

	err = json.Unmarshal([]byte(str), &value)
	if err != nil {
		return false, err
	}

	return loaded, nil
}

func (s *JSON) CompareAndDelete(ctx context.Context, key string, old any) (deleted bool, err error) {
	b, err := json.Marshal(old)
	if err != nil {
		return false, err
	}

	return s.Cache.CompareAndDelete(ctx, key, b)
}

func (s *JSON) CompareAndSwap(ctx context.Context, key string, old, value any, ttl time.Duration) (swapped bool, err error) {
	a, err := json.Marshal(old)
	if err != nil {
		return false, err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	return s.Cache.CompareAndSwap(ctx, key, a, b, ttl)
}
