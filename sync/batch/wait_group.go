package batch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/alextanhongpin/core/sync/promise"
)

type loader[K comparable, V any] interface {
	LoadManyResult(ctx context.Context, ks []K) (map[K]*Result[V], error)
}

type WaitGroup[K comparable, V any] struct {
	loader loader[K, V]
	keys   []K
	group  *promise.Group[V]
	begin  sync.WaitGroup
	end    sync.WaitGroup
}

func NewWaitGroup[K comparable, V any](l loader[K, V]) *WaitGroup[K, V] {
	return &WaitGroup[K, V]{
		loader: l,
		group:  promise.NewGroup[V](),
	}
}

// Add n number of calls to Load and/or LoadMany, NOT the number of keys.
func (wg *WaitGroup[K, V]) Add(n int) {
	wg.begin.Add(n)
	wg.end.Add(n)
}

func (wg *WaitGroup[K, V]) Load(key K) (v V, err error) {
	p, loaded := wg.group.LoadOrStore(fmt.Sprint(key))
	if !loaded {
		wg.keys = append(wg.keys, key)
	}
	wg.begin.Done()
	defer wg.end.Done()

	return p.Await()
}

func (wg *WaitGroup[K, V]) LoadMany(keys []K) (v []V, err error) {
	for _, key := range keys {
		_, loaded := wg.group.LoadOrStore(fmt.Sprint(key))
		if !loaded {
			wg.keys = append(wg.keys, key)
		}
	}

	wg.begin.Done()
	defer wg.end.Done()

	vs := make([]V, 0, len(keys))
	for _, key := range keys {
		p, _ := wg.group.Load(fmt.Sprint(key))
		v, err := p.Await()
		if errors.Is(err, ErrKeyNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}

	return vs, nil
}

func (wg *WaitGroup[K, V]) Wait(ctx context.Context) error {
	wg.begin.Wait()     // Wait for all other goroutines to signal.
	defer wg.end.Wait() // Wait for all other goroutines to complete.

	keys := slices.Clone(wg.keys)
	clear(wg.keys)
	m, err := wg.loader.LoadManyResult(ctx, keys)
	if err != nil {
		return err
	}

	for k, v := range m {
		_, _ = wg.group.Do(fmt.Sprint(k), v.Unwrap)
	}

	return nil
}
