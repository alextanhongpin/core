package cache

import (
	"context"
	"encoding/json"
	"errors"
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
	b, err := s.Cache.Load(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &v)
}

func (s *JSON) Store(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Cache.Store(ctx, key, b, ttl)
}

func (s *JSON) LoadAndDelete(ctx context.Context, key string, value any) error {
	b, err := s.Cache.LoadAndDelete(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &value)
}

func (s *JSON) CompareAndDelete(ctx context.Context, key string, old any) error {
	b, err := json.Marshal(old)
	if err != nil {
		return err
	}

	return s.Cache.CompareAndDelete(ctx, key, b)
}

func (s *JSON) CompareAndSwap(ctx context.Context, key string, old, value any, ttl time.Duration) error {
	a, err := json.Marshal(old)
	if err != nil {
		return err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Cache.CompareAndSwap(ctx, key, a, b, ttl)
}

func (s *JSON) LoadOrStore(ctx context.Context, key string, value any, getter func() (any, error), ttl time.Duration) (loaded bool, err error) {
	err = s.Load(ctx, key, &value)
	// Loaded, return early if the value exists.
	if err == nil {
		return true, nil
	}
	// If the error is not ErrNotExist, return the error.
	if !errors.Is(err, ErrNotExist) {
		return false, err
	}

	v, err := getter()
	if err != nil {
		return false, err
	}

	b, err := json.Marshal(v)
	if err != nil {
		return false, err
	}

	curr, loaded, err := s.Cache.LoadOrStore(ctx, key, b, ttl)
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(curr, &value); err != nil {
		return false, err
	}

	return loaded, nil
}
