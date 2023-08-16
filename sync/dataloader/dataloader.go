package dataloader

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/maps"
)

var (
	ErrKeyNotFound = errors.New("dataloader: key not found")
	ErrKeyRejected = errors.New("dataloader: key rejected")
)

type Promise[T any] interface {
	Await() (T, error)
}

type Option[K comparable, V any] struct {
	BatchMaxKeys int
	BatchTimeout time.Duration
	IdleTimeout  time.Duration
	BatchFn      batchFunc[K, V]
	KeyFn        keyFunc[K, V]
}

type DataLoader[K comparable, V any] struct {
	cache        map[K]*thunk[V]
	ch           chan K
	mu           sync.Mutex
	wg           sync.WaitGroup
	awake        atomic.Bool
	end          sync.Once
	batchMaxKeys int
	batchTimeout time.Duration
	batchFn      batchFunc[K, V]
	idleTimeout  time.Duration
	keyFn        keyFunc[K, V]
	done         chan struct{}
}

func New[K comparable, V any](opt Option[K, V]) *DataLoader[K, V] {
	if opt.BatchMaxKeys <= 0 {
		opt.BatchMaxKeys = 1_000
	}

	if opt.BatchTimeout == 0 {
		opt.BatchTimeout = 1 * time.Millisecond
	}

	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = 10 * opt.BatchTimeout
	}

	if opt.IdleTimeout < opt.BatchTimeout {
		panic("dataloader: idle timeout must be greater than batch timeout")
	}

	if opt.BatchFn == nil {
		panic("dataloader: missing option BatchFn in constructor")
	}

	if opt.KeyFn == nil {
		panic("dataloader: missing option KeyFn in constructor")
	}

	return &DataLoader[K, V]{
		cache:        make(map[K]*thunk[V]),
		done:         make(chan struct{}),
		ch:           make(chan K),
		batchMaxKeys: opt.BatchMaxKeys,
		batchTimeout: opt.BatchTimeout,
		batchFn:      opt.BatchFn,
		keyFn:        opt.KeyFn,
		idleTimeout:  opt.IdleTimeout,
	}
}

// Set sets the key-value after expiring existing references.
func (d *DataLoader[K, V]) Set(ctx context.Context, k K, v V) {
	d.mu.Lock()
	e, ok := d.cache[k]
	if ok {
		e.reject(KeyRejectedError(k))
	}

	t := newThunk[V]()
	t.resolve(v)
	d.cache[k] = t

	d.mu.Unlock()
}

// SetNX sets the key-value, only if the entry does not exists.
// This prevents issue with references.
func (d *DataLoader[K, V]) SetNX(ctx context.Context, k K, v V) bool {
	d.mu.Lock()
	_, ok := d.cache[k]
	if ok {
		d.mu.Unlock()
		return false
	}

	t := newThunk[V]()
	t.resolve(v)
	d.cache[k] = t

	d.mu.Unlock()

	return true
}

func (d *DataLoader[K, V]) Load(ctx context.Context, k K) Promise[V] {
	_ = d.wake(ctx)

	d.mu.Lock()
	v, ok := d.cache[k]
	if ok {
		d.mu.Unlock()

		return v
	}

	t := newThunk[V]()
	d.cache[k] = t

	d.mu.Unlock()

	select {
	case <-d.done:
		t.reject(KeyRejectedError(k))
	case d.ch <- k:
	}

	return t
}

func (d *DataLoader[K, V]) LoadMany(ctx context.Context, ks []K) Promises[V] {
	if len(ks) == 0 {
		return nil
	}

	_ = d.wake(ctx)

	result := make([]Promise[V], len(ks))

	d.mu.Lock()

	for i, k := range ks {
		v, ok := d.cache[k]
		if ok {
			result[i] = v
		} else {
			t := newThunk[V]()
			d.cache[k] = t
			result[i] = t

			select {
			case <-d.done:
				t.reject(KeyRejectedError(k))
			case d.ch <- k:
			}
		}
	}

	d.mu.Unlock()

	return result
}

func (d *DataLoader[K, V]) Stop() {
	d.stop()
}

func (d *DataLoader[K, V]) wake(ctx context.Context) bool {
	if d.isDone() {
		return false
	}

	if d.awake.Swap(true) {
		return false
	}

	d.start(ctx)
	return true
}

func (d *DataLoader[K, V]) isDone() bool {
	select {
	case <-d.done:
		return true
	default:
		return false
	}
}

func (d *DataLoader[K, V]) start(ctx context.Context) {
	d.wg.Add(1)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer d.wg.Done()
		defer cancel()
		defer d.awake.Swap(false)

		d.loop(ctx)
	}()
}

func (d *DataLoader[K, V]) stop() {
	d.end.Do(func() {
		close(d.done)

		d.wg.Wait()
	})
}

func (d *DataLoader[K, V]) loop(ctx context.Context) {
	tick := time.NewTicker(d.batchTimeout)
	defer tick.Stop()

	idle := time.NewTicker(d.idleTimeout)
	defer idle.Stop()

	keys := make(map[K]struct{})

	fetch := func() []K {
		k := maps.Keys(keys)
		clear(keys)

		return k
	}

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
		case <-d.done:
			return
		case <-idle.C:
			return
		case k := <-d.ch:
			// Two strategy for optimizing fetching data:
			// 1. Batch: when the number of keys hits the threshold
			// 2. Debounce: when no new keys after the timeout.
			keys[k] = struct{}{}
			if len(keys) >= d.batchMaxKeys {
				flush(fetch())
			}

			idle.Reset(d.idleTimeout)

			// Fire after batchTimeout.
			tick.Reset(d.batchTimeout)
		case <-tick.C:
			flush(fetch())
		}
	}
}

func (d *DataLoader[K, V]) batch(ctx context.Context, keys []K) {
	vals, err := d.batchFn(ctx, keys)
	if err != nil {
		d.mu.Lock()
		for _, k := range keys {
			d.cache[k].reject(KeyNotFoundError(k))
		}
		d.mu.Unlock()

		return
	}

	d.mu.Lock()
	for _, v := range vals {
		k, err := d.keyFn(v)
		if err != nil {
			d.cache[k].reject(err)
		} else {
			d.cache[k].resolve(v)
		}
	}

	for _, k := range keys {
		d.cache[k].reject(KeyNotFoundError(k))
	}

	d.mu.Unlock()
}

type keyFunc[K comparable, V any] func(v V) (K, error)

type batchFunc[K comparable, V any] func(ctx context.Context, keys []K) ([]V, error)

type thunk[T any] struct {
	wg    sync.WaitGroup
	begin sync.Once
	data  T
	err   error
}

func newThunk[T any]() *thunk[T] {
	t := &thunk[T]{}
	t.wg.Add(1)
	return t
}

func (t *thunk[T]) resolve(v T) {
	t.begin.Do(func() {
		t.data = v
		t.wg.Done()
	})
}

func (t *thunk[T]) reject(err error) {
	t.begin.Do(func() {
		t.err = err
		t.wg.Done()
	})
}

func (t *thunk[T]) Await() (T, error) {
	t.wg.Wait()

	return t.data, t.err
}

type Promises[T any] []Promise[T]

func (promises Promises[T]) Await() ([]T, error) {
	res := make([]T, len(promises))

	var errs error
	for i, p := range promises {
		v, err := p.Await()
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			res[i] = v
		}
	}
	if errs != nil {
		return nil, errs
	}

	return res, nil
}

func KeyNotFoundError(k any) error {
	return fmt.Errorf("%w: %s", ErrKeyNotFound, k)
}

func KeyRejectedError(k any) error {
	return fmt.Errorf("%w: %s", ErrKeyRejected, k)
}
