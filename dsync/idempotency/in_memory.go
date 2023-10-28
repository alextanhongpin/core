package idempotency

import (
	"context"
	"sync"
	"time"
)

type result[T any] struct {
	deadline time.Time
	data     data[T]
}

func (r result[T]) isExpired() bool {
	return r.deadline.Before(time.Now())
}

var _ store[any] = (*inMemoryStore[any])(nil)

type inMemoryStore[T any] struct {
	mu     sync.RWMutex
	values map[string]result[T]
}

func NewInMemoryStore[T any]() store[T] {
	return &inMemoryStore[T]{
		values: make(map[string]result[T]),
	}
}

func (s *inMemoryStore[T]) Lock(ctx context.Context, key string, lockTimeout time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, exists := s.values[key]
	if exists && !v.isExpired() {
		return false, nil
	}

	s.values[key] = result[T]{
		data: data[T]{
			Status: "started",
		},
		deadline: time.Now().Add(lockTimeout),
	}

	return true, nil
}

func (s *inMemoryStore[T]) Unlock(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()

	return nil
}

func (s *inMemoryStore[T]) Load(ctx context.Context, key string) (*data[T], error) {
	s.mu.RLock()
	v := s.values[key]
	s.mu.RUnlock()

	return &v.data, nil
}

func (s *inMemoryStore[T]) Save(ctx context.Context, key string, d data[T], duration time.Duration) error {
	s.mu.Lock()
	s.values[key] = result[T]{
		data:     d,
		deadline: time.Now().Add(duration),
	}
	s.mu.Unlock()

	return nil
}
