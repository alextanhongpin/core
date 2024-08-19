package batch

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/sync/promise"
)

type loader[K comparable, V any] interface {
	LoadManyResult(ctx context.Context, ks []K) (map[K]*Result[V], error)
}

type WaitGroup[K comparable, V any] struct {
	loader loader[K, V]
	keys   []K
	group  *promise.Group[V]
}

func NewWaitGroup[K comparable, V any](l loader[K, V]) *WaitGroup[K, V] {
	return &WaitGroup[K, V]{
		loader: l,
		group:  promise.NewGroup[V](),
	}
}

func (wg *WaitGroup[K, V]) Load(key K) (v V, err error) {
	p, loaded := wg.group.LoadOrStore(fmt.Sprint(key))
	if !loaded {
		wg.keys = append(wg.keys, key)
	}

	return p.Await()
}

func (wg *WaitGroup[K, V]) LoadMany(keys []K) (v []V, err error) {
	for _, key := range keys {
		_, loaded := wg.group.LoadOrStore(fmt.Sprint(key))
		if !loaded {
			wg.keys = append(wg.keys, key)
		}
	}

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
	defer clear(wg.keys)
	m, err := wg.loader.LoadManyResult(ctx, wg.keys)
	if err != nil {
		return err
	}

	for k, v := range m {
		_, _ = wg.group.Do(fmt.Sprint(k), v.Unwrap)
	}

	return nil
}
