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
	// ErrNoResult is returned if the key does not have a value.
	// This might not necessarily be an error, for example, if the row is not
	// found in the database when performing `SELECT ... IN (k1, k2, ... kn)`.
	ErrNoResult = errors.New("dataloader: no result")

	// ErrTerminated is returned when the dataloader is terminated.
	ErrTerminated = errors.New("dataloader: terminated")
)

type Options[K comparable, V any] struct {
	BatchFn        func(ctx context.Context, keys []K) (map[K]V, error)
	BatchMaxKeys   int
	BatchTimeout   time.Duration
	BatchQueueSize int
	Cache          cache[K, V]
}

func (o *Options[K, V]) Valid() error {
	if o.BatchFn == nil {
		return errors.New("dataloader: BatchFn is required")
	}

	o.BatchMaxKeys = cmp.Or(o.BatchMaxKeys, 1_000)
	if o.BatchMaxKeys < 1 {
		return errors.New("dataloader: BatchMaxKeys must be greater than zero")
	}

	o.BatchTimeout = cmp.Or(o.BatchTimeout, 16*time.Millisecond)
	if o.BatchTimeout < 1 {
		return errors.New("dataloader: BatchTimeout must be greater than zero")
	}

	if o.Cache == nil {
		o.Cache = NewCache[K, V]()
	}

	return nil
}

type DataLoader[K comparable, V any] struct {
	// Lifecycle control.
	begin sync.Once
	end   sync.Once

	// Concurrency primitives.
	wg sync.WaitGroup

	// State.
	pg     *promise.Group[V]
	ch     chan K
	ctx    context.Context
	cancel func(error)

	// Options.
	opts *Options[K, V]
}

// New returns a new DataLoader. The context is passed in to control the lifecycle.
func New[K comparable, V any](ctx context.Context, opts *Options[K, V]) *DataLoader[K, V] {
	if err := opts.Valid(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancelCause(ctx)
	return &DataLoader[K, V]{
		pg:     promise.NewGroup[V](),
		ch:     make(chan K),
		opts:   opts,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Set sets the key-value after expiring existing references.
// There is no use passing context as the first argument as it does not control
// the lifecycle.
func (d *DataLoader[K, V]) Set(k K, v V) {
	d.opts.Cache.Set(k, v, nil)
}

func (d *DataLoader[K, V]) Load(k K) (V, error) {
	return d.load(k).Await()
}

func (d *DataLoader[K, V]) LoadMany(ks []K) ([]promise.Result[V], error) {
	res := make(promise.Promises[V], len(ks))
	for i, k := range ks {
		res[i] = d.load(k)
	}

	return res.AllSettled(), nil
}

func (d *DataLoader[K, V]) Stop() {
	d.end.Do(func() {
		// Make sure the dataloader is started before stopping it.
		d.begin.Do(func() {})
		d.cancel(ErrTerminated)

		d.wg.Wait()
	})
}

func (d *DataLoader[K, V]) start(ctx context.Context) {
	d.begin.Do(func() {
		d.wg.Add(1)

		go func() {
			defer d.wg.Done()

			d.loop(ctx)
		}()
	})
}

func (d *DataLoader[K, V]) load(k K) *promise.Promise[V] {
	ctx := d.ctx
	d.start(ctx)

	v, err := d.opts.Cache.Get(k)
	if err == nil {
		return promise.Resolve(v)
	}

	// Only fetch from the db if the cache returns
	// ErrNotExist.
	// The cache can return ErrNoResult if the intended
	// cached value is nil.
	if !errors.Is(err, ErrNotExist) {
		return promise.Reject[V](err)
	}

	p, loaded := d.pg.LoadOrStore(fmt.Sprint(k))
	if loaded {
		return p
	}

	select {
	case <-ctx.Done():
		err := newKeyError(k, context.Cause(ctx))
		d.opts.Cache.Set(k, v, err)

		// Remove the key.
		d.pg.DoAndForget(fmt.Sprint(k), func() (V, error) {
			var v V
			return v, err
		})
	case d.ch <- k:
	}

	return p
}

func (d *DataLoader[K, V]) loop(ctx context.Context) {
	p1 := pipeline.Context(d.ctx, d.ch)
	p2 := pipeline.Batch(d.opts.BatchMaxKeys, d.opts.BatchTimeout, p1)
	p3 := pipeline.Queue(d.opts.BatchQueueSize, p2)

	for vs := range p3 {
		d.batch(d.ctx, vs)
	}
}

func (d *DataLoader[K, V]) batch(ctx context.Context, keys []K) {
	kv, err := d.opts.BatchFn(ctx, keys)
	for _, k := range keys {
		fn := func() (V, error) {
			if err != nil {
				var v V
				return v, newKeyError(k, err)
			}

			v, ok := kv[k]
			if ok {
				return v, nil
			}

			return v, newKeyError(k, ErrNoResult)
		}
		_, _ = d.pg.DoAndForget(fmt.Sprint(k), func() (V, error) {
			v, err := fn()
			d.opts.Cache.Set(k, v, err)
			return v, err
		})
	}
}

func newKeyError(k any, err error) *KeyError {
	return &KeyError{
		Key: fmt.Sprint(k),
		err: err,
	}
}

type KeyError struct {
	Key string
	err error
}

func (e *KeyError) Error() string {
	return fmt.Sprintf("dataloader.KeyError(%s): %s", e.Key, e.err)
}

func (e *KeyError) Is(err error) bool {
	return errors.Is(e.err, err)
}

func (e *KeyError) Unwrap() error {
	return e.err
}
