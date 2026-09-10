package dataloader

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/cache"
	"github.com/alextanhongpin/core/sync/pipeline"
)

var (
	ErrNotFound = errors.New("dataloader: not found")
	ErrCanceled = errors.New("dataloader: canceled")
)

type batchFn[K comparable, V any] = func(ctx context.Context, keys []K) (map[K]V, error)

type DataLoader[K comparable, V any] struct {
	*Config
	batchFn batchFn[K, V]
	ch      chan K
	ctx     context.Context
	cache   *cache.Cache[K, future[V]]
}

type Config struct {
	BatchInterval time.Duration
	BatchSize     int
	BufferSize    int
}

func DefaultConfig() *Config {
	return &Config{
		BatchInterval: 16 * time.Millisecond,
		BatchSize:     25,
	}
}

func New[K comparable, V any](ctx context.Context, fn batchFn[K, V], cfg *Config) (*DataLoader[K, V], func()) {
	cfg = cmp.Or(cfg, DefaultConfig())
	ctx, cancel := context.WithCancelCause(ctx)
	dl := &DataLoader[K, V]{
		Config:  cfg,
		batchFn: fn,
		ch:      make(chan K, cfg.BufferSize),
		ctx:     ctx,
		cache: cache.New(func(key K) (*future[V], error) {
			return newFuture[V](ctx), nil
		}),
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		dl.background(ctx)
	})

	return dl, sync.OnceFunc(func() {
		cancel(ErrCanceled)

		wg.Wait()
	})
}

type Result[K comparable, V any] struct {
	Key   K
	Value V
	Error error
}

func (d *DataLoader[K, V]) background(ctx context.Context) {
	p1 := pipeline.SourceChan(ctx, d.ch)
	p2 := pipeline.Batch(p1, d.BatchSize, d.BatchInterval)
	pipeline.Sink(p2, func(keys []K) {
		res, err := d.batchFn(ctx, keys)
		if err != nil {
			// All keys becomes error.
			for _, key := range keys {
				f, loaded, _ := d.cache.LoadOrCreate(key)
				if !loaded {
					panic("lost reference to strong pointer")
				}
				f.Reject(err)
			}
			return
		}
		for _, k := range keys {
			f, loaded, _ := d.cache.LoadOrCreate(k)
			if !loaded {
				panic("lost reference to strong pointer")
			}
			if v, ok := res[k]; ok {
				f.Resolve(v)
			} else {
				// Key not found
				f.Reject(fmt.Errorf("%w: %v", ErrNotFound, k))
			}
		}
	})
}

func (d *DataLoader[K, V]) load(key K) (*future[V], error) {
	select {
	case <-d.ctx.Done():
		return nil, context.Cause(d.ctx)

	default:
		fut, loaded, _ := d.cache.LoadOrCreate(key)
		if loaded {
			return fut, nil
		}

		select {
		case <-d.ctx.Done():
			fut.Reject(context.Cause(d.ctx))
		case d.ch <- key:
		}

		return fut, nil
	}
}

func (d *DataLoader[K, V]) Load(key K) (V, error) {
	fut, err := d.load(key)
	if err != nil {
		var zero V
		return zero, err
	}
	return fut.Wait()
}

func (d *DataLoader[K, V]) LoadMany(keys ...K) ([]*Result[K, V], error) {
	select {
	case <-d.ctx.Done():
		return nil, context.Cause(d.ctx)

	default:
		fs := make([]*future[V], len(keys))
		for i, key := range keys {
			f, err := d.load(key)
			if err != nil {
				return nil, err
			}
			fs[i] = f
		}
		res := make([]*Result[K, V], len(keys))
		for i, f := range fs {
			v, err := f.Wait()
			res[i] = &Result[K, V]{
				Key:   keys[i],
				Value: v,
				Error: err,
			}
		}
		return res, nil
	}
}

type future[T any] struct {
	ctx    context.Context
	cancel func(error)
}

func newFuture[T any](ctx context.Context) *future[T] {
	ctx, cancel := context.WithCancelCause(ctx)

	return &future[T]{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (f *future[T]) Reject(err error) {
	f.cancel(err)
}

func (f *future[T]) Resolve(val T) {
	f.cancel(&errVal[T]{val})
}

func (f *future[T]) Wait() (T, error) {
	<-f.ctx.Done()
	err := context.Cause(f.ctx)
	if e, ok := errors.AsType[*errVal[T]](err); ok {
		return e.val, nil
	}
	var zero T
	return zero, err
}

type errVal[T any] struct {
	val T
}

func (e *errVal[T]) Error() string {
	return ""
}
