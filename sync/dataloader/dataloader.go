package dataloader

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
	"golang.org/x/exp/maps"
)

var (
	// ErrNoResult is returned if the key does not have a value.
	// This might not necessarily be an error, for example, if the row is not
	// found in the database when performing `SELECT ... IN (k1, k2, ... kn)`.
	ErrNoResult = errors.New("no result")

	// ErrAborted is returned when the operation is cancelled or replaced with
	// another running process.
	ErrAborted = errors.New("aborted")

	// ErrTerminated is returned when the dataloader is terminated.
	ErrTerminated = errors.New("terminated")
)

type Options[K comparable, V any] struct {
	BatchFn      batchFunc[K, V]
	BatchMaxKeys int
	BatchTimeout time.Duration
	// KeyFn maps the result back to the key.
	// This is necessary, because the results from the batch function may not be
	// in the same order as the keys, or may not even exists.
	KeyFn      keyFunc[K, V]
	NoDebounce bool
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

	if o.KeyFn == nil {
		return errors.New("dataloader: KeyFn is required")
	}

	return nil
}

type DataLoader[K comparable, V any] struct {
	// Lifecycle control.
	begin sync.Once
	end   sync.Once

	// Concurrency primitives.
	mu sync.Mutex
	wg sync.WaitGroup

	// State.
	cache  map[K]*promise.Promise[V]
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
		cache:  make(map[K]*promise.Promise[V]),
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
	d.mu.Lock()
	e, ok := d.cache[k]
	if ok {
		// Reject the running promise so that the caller will stop waiting.
		e.Reject(newKeyError(k, ErrAborted))
	}

	d.cache[k] = promise.Resolve(v)
	d.mu.Unlock()
}

// SetNX sets the key-value, only if the entry does not exists.
// This prevents issue with references.
func (d *DataLoader[K, V]) SetNX(k K, v V) bool {
	d.mu.Lock()
	_, ok := d.cache[k]
	if ok {
		d.mu.Unlock()
		return false
	}

	d.cache[k] = promise.Resolve(v)
	d.mu.Unlock()

	return true
}

func (d *DataLoader[K, V]) Load(k K) (V, error) {
	select {
	case <-d.ctx.Done():
		var v V
		return v, context.Cause(d.ctx)
	default:
		d.start(d.ctx)
	}

	d.mu.Lock()
	v, ok := d.cache[k]
	if ok {
		d.mu.Unlock()

		return v.Await()
	}

	p := promise.Deferred[V]()
	d.cache[k] = p
	d.mu.Unlock()

	select {
	case <-d.ctx.Done():
		// Immediately rejects.
		p.Reject(newKeyError(k, context.Cause(d.ctx)))
	case d.ch <- k:
	}

	return p.Await()
}

func (d *DataLoader[K, V]) LoadMany(ks []K) ([]promise.Result[V], error) {
	// No keys to load.
	if len(ks) == 0 {
		return nil, nil
	}

	select {
	case <-d.ctx.Done():
		return nil, context.Cause(d.ctx)
	default:
		d.start(d.ctx)
	}

	res := make(promise.Promises[V], len(ks))

	d.mu.Lock()

	for i, k := range ks {
		v, ok := d.cache[k]
		if ok {
			res[i] = v
		} else {
			p := promise.Deferred[V]()
			d.cache[k] = p

			res[i] = p

			select {
			case <-d.ctx.Done():
				p.Reject(newKeyError(k, context.Cause(d.ctx)))
			case d.ch <- k:
			}
		}
	}

	d.mu.Unlock()

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
	t := time.NewTicker(d.opts.BatchTimeout)
	defer t.Stop()

	keys := make(map[K]struct{})

	fetch := func() []K {
		k := maps.Keys(keys)
		clear(keys)

		return k
	}

	// flush clears all existing keys by doing a final batch request.
	flush := func(keys []K) {
		if len(keys) == 0 {
			return
		}

		d.wg.Add(1)

		go func() {
			defer d.wg.Done()

			d.batch(ctx, keys)
		}()
	}

	defer func() {
		flush(fetch())
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case k := <-d.ch:
			// Two strategy for optimizing fetching data:
			// 1. Batch: when the number of keys hits the threshold
			// 2. Debounce: when no new keys after the timeout.
			keys[k] = struct{}{}
			// 1.
			if len(keys) >= d.opts.BatchMaxKeys {
				flush(fetch())
			}

			if !d.opts.NoDebounce {
				// Keep debouncing until the max keys threshold is reached.
				t.Reset(d.opts.BatchTimeout)
			}
		// 2.
		case <-t.C:
			flush(fetch())
		}
	}
}

func (d *DataLoader[K, V]) batch(ctx context.Context, keys []K) {
	vals, err := d.opts.BatchFn(ctx, keys)
	if err != nil {
		d.mu.Lock()
		for _, k := range keys {
			d.cache[k].Reject(newKeyError(k, err))
		}
		d.mu.Unlock()

		return
	}

	d.mu.Lock()
	for _, v := range vals {
		k, err := d.opts.KeyFn(v)
		if err != nil {
			d.cache[k].Reject(err)
		} else {
			d.cache[k].Resolve(v)
		}
	}

	for _, k := range keys {
		d.cache[k].Reject(newKeyError(k, ErrNoResult))
	}

	d.mu.Unlock()
}

type keyFunc[K comparable, V any] func(v V) (K, error)

type batchFunc[K comparable, V any] func(ctx context.Context, keys []K) ([]V, error)

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
