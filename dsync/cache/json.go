package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

func (s *JSON) LoadOrStore(ctx context.Context, key string, value any, getter func() (any, error), ttl time.Duration) (loaded bool, err error) {
	err = s.Load(ctx, key, &value)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, ErrNotExist) {
		return false, err
	}

	v, err := getter()
	if err != nil {
		return false, err
	}

	if err := s.Store(ctx, key, v, ttl); err != nil {
		return false, err
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false, fmt.Errorf("value must be a non-nil pointer, got %T", value)
	}

	rv.Elem().Set(reflect.ValueOf(v))

	return false, nil
}
