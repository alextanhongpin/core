package batch

import (
	"errors"
	"fmt"

	"github.com/mitchellh/copystructure"
)

var (
	ErrKeyNotFound = errors.New("batch: key not found")
	ErrClosed      = errors.New("batch: already closed")
)

type BatchFn[K comparable, V any] func(...K) ([]V, error)

type KeyFn[K comparable, V any] func(V) (K, error)

type Loader[K comparable, V any] struct {
	keys    []K
	vals    []*V
	batchFn BatchFn[K, V]
	keyFn   KeyFn[K, V]
	done    chan bool
}

func NewLoader[K comparable, V any](
	batchFn BatchFn[K, V],
	keyFn KeyFn[K, V],
) *Loader[K, V] {
	return &Loader[K, V]{
		batchFn: batchFn,
		keyFn:   keyFn,
		done:    make(chan bool),
	}
}

func (l *Loader[K, V]) Load(k K) *V {
	if err := l.closed(); err != nil {
		panic(err)
	}

	return l.load(k)
}

func (l *Loader[K, V]) Wait() error {
	if err := l.closed(); err != nil {
		return err
	}

	defer close(l.done)
	return l.wait()
}

func (l *Loader[K, V]) closed() error {
	select {
	case <-l.done:
		return ErrClosed
	default:
		return nil
	}
}

func (l *Loader[K, V]) load(k K) *V {
	l.keys = append(l.keys, k)
	v := new(V)
	l.vals = append(l.vals, v)
	return v
}

func (l *Loader[K, V]) wait() error {
	// Don't trigger batchFn if there are no keys.
	if len(l.keys) == 0 {
		return nil
	}

	vals, err := l.batchFn(l.keys...)
	if err != nil {
		return err
	}

	valByKey := make(map[K]V)
	for _, v := range vals {
		v := v

		k, err := l.keyFn(v)
		if err != nil {
			return err
		}

		valByKey[k] = v
	}

	cached := make(map[K]bool)
	for i, k := range l.keys {
		v, ok := valByKey[k]
		if !ok {
			return fmt.Errorf("%w: %v", ErrKeyNotFound, k)
		}

		if cached[k] {
			// If there are duplicate keys, clone the subsequent value.
			// This prevents sharing reference for the same value, which is a common
			// mistake.
			c, err := copystructure.Copy(v)
			if err != nil {
				return err
			}

			*l.vals[i] = c.(V)
		} else {
			*l.vals[i] = v
			cached[k] = true
		}
	}

	return nil
}
