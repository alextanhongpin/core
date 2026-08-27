package dataloader

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
	"github.com/alextanhongpin/core/sync/promise"
)

var (
	ErrNotFound = errors.New("dataloader: not found")
	ErrCanceled = errors.New("dataloader: canceled")
)

type batchFn[K comparable, V any] = func(ctx context.Context, keys []K) (map[K]V, error)

type DataLoader[K comparable, V any] struct {
	batchFn batchFn[K, V]
	ch      chan K
	ctx     context.Context
	pm      *promise.Map[K, V]
	wg      sync.WaitGroup
	Config
}

type Config struct {
	BatchSize     int
	BatchInterval time.Duration
}

func New[K comparable, V any](ctx context.Context, fn batchFn[K, V], cfg Config) (*DataLoader[K, V], func()) {
	ctx, cancel := context.WithCancelCause(ctx)
	dl := &DataLoader[K, V]{
		batchFn: fn,
		ctx:     ctx,
		ch:      make(chan K),
		pm:      promise.NewMap[K, V](),
		Config:  cfg,
	}
	dl.wg.Go(func() {
		dl.background(ctx)
	})
	return dl, sync.OnceFunc(func() {
		cancel(ErrCanceled)
		dl.wg.Wait()
		dl.pm.Range(func(key, val any) bool {
			dl.pm.Reject(context.Background(), key.(K), ErrCanceled)
			return true
		})
	})
}

type Result[K comparable, V any] struct {
	Key   K
	Value V
	Error error
}

func (d *DataLoader[K, V]) background(ctx context.Context) {
	p1 := pipeline.SourceChan(ctx, d.ch)
	p2 := pipeline.Dedup(p1)
	p3 := pipeline.Batch(p2, d.BatchSize, d.BatchInterval)
	pipeline.Sink(p3, func(keys []K) {
		res, err := d.batchFn(ctx, keys)
		if err != nil {
			// All keys becomes error.
			for _, key := range keys {
				d.pm.Reject(ctx, key, err)
			}
			return
		}
		for _, k := range keys {
			if v, ok := res[k]; ok {
				d.pm.Resolve(ctx, k, v)
			} else {
				// Key not found
				d.pm.Reject(ctx, k, fmt.Errorf("%w: key=%v", ErrNotFound, k))
			}
		}
	})
}

func (d *DataLoader[K, V]) load(key K) *promise.Promise[V] {
	p := d.pm.Defer(d.ctx, key)

	select {
	case <-d.ctx.Done():
		return p
	case d.ch <- key:
	}

	return p
}

func (d *DataLoader[K, V]) Delete(key K) {
	d.pm.Delete(d.ctx, key, ErrCanceled)
}

func (d *DataLoader[K, V]) Load(key K) (V, error) {
	return d.load(key).Await()
}

func (d *DataLoader[K, V]) LoadMany(keys ...K) ([]*Result[K, V], error) {
	ps := make([]*promise.Promise[V], len(keys))
	for i, key := range keys {
		ps[i] = d.load(key)
	}
	res := make([]*Result[K, V], len(keys))
	for i, p := range ps {
		v, err := p.Await()
		res[i] = &Result[K, V]{
			Key:   keys[i],
			Value: v,
			Error: err,
		}
	}
	return res, nil
}
