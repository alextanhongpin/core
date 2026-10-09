package dataloader

import (
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

// DataLoader batches concurrent loads. Cached results are weakly held and may
// be loaded again after garbage collection.
type DataLoader[K comparable, V any] struct {
	// Config is a snapshot; changing it does not reconfigure the loader.
	*Config
	batchFn batchFn[K, V]
	ch      chan request[K, V]
	cfg     Config
	ctx     context.Context
	mu      sync.Mutex
	wg      sync.WaitGroup
	stopped bool
	cache   *cache.Cache[K, future[V]]
}

type request[K comparable, V any] struct {
	key    K
	future *future[V]
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

// New starts a batch worker. Zero batch size and interval use defaults.
// Invalid configuration or a nil function panics before work starts.
// The returned idempotent stop function cancels pending loads and waits for
// the worker; the batch function must honor cancellation for prompt shutdown.
func New[K comparable, V any](ctx context.Context, fn batchFn[K, V], cfg *Config) (*DataLoader[K, V], func()) {
	effective := *DefaultConfig()
	if cfg != nil {
		effective = *cfg
		if effective.BatchInterval == 0 {
			effective.BatchInterval = DefaultConfig().BatchInterval
		}
		if effective.BatchSize == 0 {
			effective.BatchSize = DefaultConfig().BatchSize
		}
	}
	if effective.BatchInterval < 0 || effective.BatchSize < 0 || effective.BufferSize < 0 {
		panic("dataloader: invalid configuration")
	}
	if fn == nil {
		panic("dataloader: nil batch function")
	}
	// Keep the public configuration snapshot separate from operational state.
	snapshot := effective
	ctx, cancel := context.WithCancelCause(ctx)
	dl := &DataLoader[K, V]{
		Config:  &snapshot,
		cfg:     effective,
		batchFn: fn,
		ch:      make(chan request[K, V], effective.BufferSize),
		ctx:     ctx,
		cache: cache.New(func(key K) (*future[V], error) {
			return newFuture[V](ctx), nil
		}),
	}

	dl.wg.Go(func() {
		dl.background(ctx)
	})

	return dl, sync.OnceFunc(func() {
		dl.mu.Lock()
		dl.stopped = true
		cancel(ErrCanceled)
		dl.mu.Unlock()
		dl.wg.Wait()
	})
}

type Result[K comparable, V any] struct {
	Key   K
	Value V
	Error error
}

func (d *DataLoader[K, V]) background(ctx context.Context) {
	p1 := pipeline.SourceChan(ctx, d.ch)
	p2 := pipeline.Batch(p1, d.cfg.BatchSize, d.cfg.BatchInterval)
	pipeline.Sink(p2, func(requests []request[K, V]) {
		if ctx.Err() != nil {
			return
		}
		keys := make([]K, len(requests))
		for i, req := range requests {
			keys[i] = req.key
		}
		res, err := d.batchFn(ctx, keys)
		for _, req := range requests {
			if err != nil {
				req.future.Reject(err)
			} else if v, ok := res[req.key]; ok {
				req.future.Resolve(v)
			} else {
				req.future.Reject(fmt.Errorf("%w: %v", ErrNotFound, req.key))
			}
		}
	})
}

func (d *DataLoader[K, V]) load(key K, async bool) (*future[V], error) {
	select {
	case <-d.ctx.Done():
		return nil, context.Cause(d.ctx)

	default:
		fut, loaded, _ := d.cache.LoadOrCreate(key)
		if loaded {
			return fut, nil
		}

		if !async {
			select {
			case <-d.ctx.Done():
				fut.Reject(context.Cause(d.ctx))
			case d.ch <- request[K, V]{key: key, future: fut}:
			}
			return fut, nil
		}
		d.mu.Lock()
		if !d.stopped {
			// Admission belongs to the loader, not to any individual waiter.
			d.wg.Go(func() {
				select {
				case <-d.ctx.Done():
					fut.Reject(context.Cause(d.ctx))
				case d.ch <- request[K, V]{key: key, future: fut}:
				}
			})
		}
		d.mu.Unlock()

		return fut, nil
	}
}

func (d *DataLoader[K, V]) Load(key K) (V, error) {
	fut, err := d.load(key, false)
	if err != nil {
		var zero V
		return zero, err
	}
	return fut.Wait()
}

// LoadContext cancels only this caller's result wait. Shared admission and
// loading continue under the loader lifetime context. Cancellation racing with
// a completed result may return either outcome.
func (d *DataLoader[K, V]) LoadContext(ctx context.Context, key K) (V, error) {
	if ctx.Err() != nil {
		var zero V
		return zero, context.Cause(ctx)
	}
	fut, err := d.load(key, true)
	if err != nil {
		var zero V
		return zero, err
	}
	return fut.WaitContext(ctx)
}

func (d *DataLoader[K, V]) LoadMany(keys ...K) ([]*Result[K, V], error) {
	select {
	case <-d.ctx.Done():
		return nil, context.Cause(d.ctx)

	default:
		fs := make([]*future[V], len(keys))
		for i, key := range keys {
			f, err := d.load(key, false)
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
	return f.result()
}

func (f *future[T]) WaitContext(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, context.Cause(ctx)
	case <-f.ctx.Done():
		return f.result()
	}
}

func (f *future[T]) result() (T, error) {
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
