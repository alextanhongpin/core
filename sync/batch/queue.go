package batch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/alextanhongpin/core/sync/promise"
)

type Queue[K comparable, V any] struct {
	loader loader[K, V]
	group  *promise.Group[V]

	mu   sync.Mutex
	keys map[K]struct{}
}

func NewQueue[K comparable, V any](l loader[K, V]) *Queue[K, V] {
	return &Queue[K, V]{
		loader: l,
		group:  promise.NewGroup[V](),
		keys:   make(map[K]struct{}),
	}
}

func (q *Queue[K, V]) Add(keys ...K) {
	q.mu.Lock()
	for _, key := range keys {
		q.keys[key] = struct{}{}
	}
	q.mu.Unlock()
}

func (q *Queue[K, V]) Load(key K) (v V, err error) {
	p, ok := q.group.Load(fmt.Sprint(key))
	if !ok {
		return v, newKeyError(fmt.Sprint(key), ErrKeyNotExist)
	}

	return p.Await()
}

func (q *Queue[K, V]) LoadMany(keys []K) (v []V, err error) {
	vs := make([]V, 0, len(keys))
	for _, key := range keys {
		p, loaded := q.group.Load(fmt.Sprint(key))
		if !loaded {
			continue
		}
		v, err := p.Await()
		if errors.Is(err, ErrKeyNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}

	return vs, nil
}

func (q *Queue[K, V]) Flush(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	kv := maps.Clone(q.keys)
	clear(q.keys)

	keys := make([]K, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}

	m, err := q.loader.LoadManyResult(ctx, keys)
	if err != nil {
		return err
	}

	for _, k := range keys {
		v, ok := m[k]
		if ok {
			q.group.Do(fmt.Sprint(k), v.Unwrap)
		} else {
			q.group.Do(fmt.Sprint(k), func() (v V, err error) {
				return v, newKeyError(fmt.Sprint(k), ErrKeyNotExist)
			})
		}
	}

	return nil
}
