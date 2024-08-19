package batch

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrKeyNotExist = errors.New("batch: key does not exist")

type LoaderOptions[K comparable, V any] struct {
	Cache   cache[K, *Result[V]]
	BatchFn func([]K) (map[K]V, error)
	TTL     time.Duration
}

func (o *LoaderOptions[K, V]) Valid() error {
	o.TTL = cmp.Or(o.TTL, time.Hour)
	if o.TTL <= 0 {
		return errors.New("batch: TTL must be greater than 0")
	}
	if o.BatchFn == nil {
		return errors.New("batch: BatchFn is required")
	}

	if o.Cache == nil {
		o.Cache = NewCache[K, *Result[V]]()
	}

	return nil
}

type Loader[K comparable, V any] struct {
	opts *LoaderOptions[K, V]
}

func NewLoader[K comparable, V any](opts *LoaderOptions[K, V]) *Loader[K, V] {
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	return &Loader[K, V]{
		opts: opts,
	}
}

func (l *Loader[K, V]) Load(ctx context.Context, k K) (v V, err error) {
	m, err := l.LoadManyResult(ctx, []K{k})
	if err != nil {
		return v, err
	}

	return m[k].Unwrap()
}

func (l *Loader[K, V]) LoadMany(ctx context.Context, ks []K) ([]V, error) {
	m, err := l.LoadManyResult(ctx, ks)
	if err != nil {
		return nil, err
	}
	res := make([]V, 0, len(ks))
	for _, k := range ks {
		r, ok := m[k]
		if !ok {
			continue
		}
		v, err := r.Unwrap()
		if errors.Is(err, ErrKeyNotExist) {
			continue
		}
		res = append(res, v)
	}

	return res, nil
}

func (l *Loader[K, V]) LoadManyResult(ctx context.Context, ks []K) (map[K]*Result[V], error) {
	m, err := l.opts.Cache.LoadMany(ctx, ks...)
	if err != nil {
		return nil, err
	}

	pks := make([]K, 0, len(ks))
	res := make(map[K]*Result[V])
	for _, k := range ks {
		if v, ok := m[k]; ok {
			res[k] = v
			continue
		}
		pks = append(pks, k)
	}
	// All keys found in cache, return.
	if len(res) == len(ks) {
		return res, nil
	}

	// Fetch the pending keys.
	b, err := l.opts.BatchFn(pks)
	if err != nil {
		return nil, err
	}

	// Stores the new keys in the cache.
	n := make(map[K]*Result[V])
	for _, k := range pks {
		v, ok := b[k]
		if ok {
			n[k] = newResult(v, nil)
		} else {
			n[k] = newResult(v, newKeyError(fmt.Sprint(k), ErrKeyNotExist))
		}
		res[k] = n[k]
	}

	if err := l.opts.Cache.StoreMany(ctx, n, l.opts.TTL); err != nil {
		return nil, err
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

type KeyError struct {
	Key string
	Err error
}

func newKeyError(key string, err error) *KeyError {
	return &KeyError{
		Key: key,
		Err: err,
	}
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("%s: %q", e.Err, e.Key)
}

func (e *KeyError) Is(err error) bool {
	return errors.Is(e.Err, err)
}
