package cmap

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrNotExist = errors.New("cmap: key does not exist")

type batchMap[K comparable, V any] struct {
	mu      sync.Mutex
	batchFn func([]K) ([]V, error)
	keyFn   func(V) (K, error)
	cache   *ttlMap[K, *Result[V]]
	ttl     time.Duration
	//cacheNoResult bool // TODO: Make this optional
}

func (b *batchMap[K, V]) Load(k K) (V, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r, ok := b.cache.load(k)
	if ok {
		return r.Unwrap()
	}

	ks := []K{k}
	vs, err := b.batchFn(ks)
	if err != nil {
		var v V
		return v, err
	}

	for _, v := range vs {
		kk, err := b.keyFn(v)
		if err != nil {
			var v V
			return v, err
		}
		if k == kk {
			b.cache.store(k, newResult(v, nil), b.ttl)
			return v, nil
		}
	}

	var v V
	return v, fmt.Errorf("%w: %v", ErrNotExist, k)
}

func (b *batchMap[K, V]) LoadMany(ks []K) ([]*Result[V], error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var pks []K

	m := make(map[K]*Result[V])
	for _, k := range ks {
		v, ok := b.cache.load(k)
		if ok {
			m[k] = v
			continue
		}

		pks = append(pks, k)
	}

	vs, err := b.batchFn(pks)
	if err != nil {
		return nil, err
	}

	for _, v := range vs {
		kk, err := b.keyFn(v)
		if err != nil {
			continue
		}
		r := newResult(v, err)
		m[kk] = r
		b.cache.store(kk, r, b.ttl)
	}

	res := make([]*Result[V], len(ks))
	for i, k := range ks {
		r, ok := m[k]
		if ok {
			res[i] = r
			continue
		}

		var v V
		r = newResult(v, fmt.Errorf("%w: %v", ErrNotExist, k))
		b.cache.store(k, r, b.ttl/2)
		res[i] = r
	}

	return res, nil
}

type Result[T any] struct {
	data T
	err  error
}

func (r *Result[T]) Unwrap() (T, error) {
	return r.data, r.err
}

func newResult[T any](data T, err error) *Result[T] {
	return &Result[T]{
		data: data,
		err:  err,
	}
}
