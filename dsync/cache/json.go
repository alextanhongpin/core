package cache

import (
	"context"
	"encoding/json"
	"time"
)

type JSON struct {
	cache *Cacheable
}

func NewJSON(cache *Cacheable) *JSON {
	return &JSON{
		cache: cache,
	}
}

func (s *JSON) Load(ctx context.Context, key string, v any) error {
	str, err := s.cache.Load(ctx, key)
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

	return s.cache.Store(ctx, key, string(b), ttl)
}

func (s *JSON) LoadAndDelete(ctx context.Context, key string, value any) (loaded bool, err error) {
	str, loaded, err := s.cache.LoadAndDelete(ctx, key)
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

	return s.cache.CompareAndDelete(ctx, key, string(b))
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

	return s.cache.CompareAndSwap(ctx, key, string(a), string(b), ttl)
}
