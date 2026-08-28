package dataloader

import (
	"cmp"
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
	*Config
	batchFn batchFn[K, V]
	ch      chan K
	ctx     context.Context
	pm      *promise.Map[K, V]
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
		pm:      promise.NewMap[K, V](ctx),
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		dl.background(ctx)
	})

	return dl, sync.OnceFunc(func() {
		cancel(ErrCanceled)

		wg.Wait()

		dl.pm.Range(func(key, val any) bool {
			dl.Delete(key.(K), ErrCanceled)
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
	p2 := pipeline.Batch(p1, d.BatchSize, d.BatchInterval)
	pipeline.Sink(p2, func(keys []K) {
		res, err := d.batchFn(ctx, keys)
		if err != nil {
			// All keys becomes error.
			for _, key := range keys {
				p, loaded, _ := d.pm.LoadOrCreate(key)
				if !loaded {
					panic("lost reference to strong pointer")
				}
				p.Reject(err)
			}
			return
		}
		for _, k := range keys {
			p, loaded, _ := d.pm.LoadOrCreate(k)
			if !loaded {
				panic("lost reference to strong pointer")
			}
			if v, ok := res[k]; ok {
				p.Resolve(v)
			} else {
				// Key not found
				p.Reject(fmt.Errorf("%w: %v", ErrNotFound, k))
			}
		}
	})
}

func (d *DataLoader[K, V]) load(key K) *promise.Promise[V] {
	p, loaded, _ := d.pm.LoadOrCreate(key)
	if loaded {
		return p
	}

	select {
	case <-d.ctx.Done():
		p.Reject(context.Cause(d.ctx))
		return p

	case d.ch <- key:
	}

	return p
}

func (d *DataLoader[K, V]) Delete(key K, err error) {
	p, loaded := d.pm.LoadAndDelete(key)
	if loaded {
		p.Reject(err)
	}
	p.Await()
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
