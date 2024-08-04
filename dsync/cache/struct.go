package cache

import (
	"context"
	"encoding/json"
	"time"
)

type Struct[T any] struct {
	cache *Cacheable
}

func NewStruct[T any](cache *Cacheable) *Struct[T] {
	return &Struct[T]{
		cache: cache,
	}
}

func (s *Struct[T]) Load(ctx context.Context, key string) (v T, err error) {
	str, err := s.cache.Load(ctx, key)
	if err != nil {
		return v, err
	}

	err = json.Unmarshal([]byte(str), &v)
	return v, err
}

func (s *Struct[T]) Store(ctx context.Context, key string, value T, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.cache.Store(ctx, key, string(b), ttl)
}

func (s *Struct[T]) LoadOrStore(ctx context.Context, key string, value T, ttl time.Duration) (old T, loaded bool, err error) {
	b, err := json.Marshal(value)
	if err != nil {
		return old, false, err
	}

	str, loaded, err := s.cache.LoadOrStore(ctx, key, string(b), ttl)
	if err != nil {
		return old, false, err
	}

	err = json.Unmarshal([]byte(str), &old)
	return old, loaded, err
}

func (s *Struct[T]) LoadAndDelete(ctx context.Context, key string) (value T, loaded bool, err error) {
	str, loaded, err := s.cache.LoadAndDelete(ctx, key)
	if err != nil {
		return value, false, err
	}
	if !loaded {
		return value, false, nil
	}

	err = json.Unmarshal([]byte(str), &value)
	return value, loaded, err
}

func (s *Struct[T]) CompareAndDelete(ctx context.Context, key string, old T) (deleted bool, err error) {
	b, err := json.Marshal(old)
	if err != nil {
		return false, err
	}

	return s.cache.CompareAndDelete(ctx, key, string(b))
}

func (s *Struct[T]) CompareAndSwap(ctx context.Context, key string, old, value T, ttl time.Duration) (swapped bool, err error) {
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
