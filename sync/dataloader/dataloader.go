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
	p := promise.Resolve(v)
	d.pg.DeleteAndStore(fmt.Sprint(k), p)
}

func (d *DataLoader[K, V]) Load(k K) (V, error) {
	ctx := d.ctx
	d.start(ctx)

	p, loaded := d.pg.LoadOrStore(fmt.Sprint(k))
	if loaded {
		return p.Await()
	}

	select {
	case <-ctx.Done():
		// Immediately rejects.
		p.Reject(newKeyError(k, context.Cause(ctx)))
	case d.ch <- k:
	}

	return p.Await()
}

func (d *DataLoader[K, V]) LoadMany(ks []K) ([]promise.Result[V], error) {
	// No keys to load.
	if len(ks) == 0 {
		return nil, nil
	}

	ctx := d.ctx
	d.start(ctx)

	res := make(promise.Promises[V], len(ks))
	for i, k := range ks {
		p, loaded := d.pg.LoadOrStore(fmt.Sprint(k))
		if loaded {
			goto labels
		}

		select {
		case <-ctx.Done():
			p.Reject(newKeyError(k, context.Cause(ctx)))
		case d.ch <- k:
		}
	labels:
		res[i] = p
	}

	return res.AllSettled(), nil
}

func (d *DataLoader[K, V]) Stop() {
	d.end.Do(func() {
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
	if err != nil {
		for _, k := range keys {
			d.pg.Do(fmt.Sprint(k), func() (V, error) {
				var v V
				return v, newKeyError(k, err)
			})
		}

		return
	}

	for _, k := range keys {
		d.pg.Do(fmt.Sprint(k), func() (V, error) {
			v, ok := kv[k]
			if ok {
				return v, nil
			}

			return v, newKeyError(k, ErrNoResult)
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
