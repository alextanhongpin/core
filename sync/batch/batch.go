// package batch simplifies data loading for one-to-one and one-to-many relationships.
package batch

import (
	"errors"
	"fmt"
	"sync"

	"github.com/alextanhongpin/core/types/sliceutil"
	"github.com/mitchellh/copystructure"
)

type kind int

const (
	zero kind = iota
	one
	many
)

var (
	ErrKeyNotFound         = errors.New("batch: key not found")
	ErrMultipleValuesFound = errors.New("batch: multiple values found")
	ErrClosed              = errors.New("batch: already closed")
	ErrNilReference        = errors.New("batch: nil reference is passed in")
)

type BatchFn[K comparable, V any] func(...K) ([]V, error)

type KeyFn[K comparable, V any] func(V) (K, error)

type Loader[K comparable, V any] struct {
	kindByID map[int]kind
	keys     []K
	one      map[int]*V
	many     map[int]*[]V
	batchFn  BatchFn[K, V]
	keyFn    KeyFn[K, V]
	done     chan bool
	mu       sync.Mutex
}

func New[K comparable, V any](batchFn BatchFn[K, V], keyFn KeyFn[K, V]) *Loader[K, V] {
	return &Loader[K, V]{
		batchFn:  batchFn,
		kindByID: make(map[int]kind),
		one:      make(map[int]*V),
		many:     make(map[int]*[]V),
		keyFn:    keyFn,
		done:     make(chan bool),
	}
}

// Load ensures that exactly one result will be loaded.
// Suitable for loading data with one-to-one
// relationships.
// Returns error if the batchFn does not return exactly 1 result.
// If the same key is loaded multiple times, the result will be deep-copied
// before returned.
func (l *Loader[K, V]) Load(v *V, k K) error {
	if v == nil {
		return ErrNilReference
	}

	if err := l.closed(); err != nil {
		return err
	}

	l.load(v, k)

	return nil
}

// LoadMany returns zero, one or many results.
// Suitable for loading data with one-to-many
// relationships.
// If the same key is loaded multiple times, the result will be deep-copied
// before returned.
func (l *Loader[K, V]) LoadMany(v *[]V, ks ...K) error {
	if v == nil {
		return ErrNilReference
	}

	if err := l.closed(); err != nil {
		return err
	}

	l.loadMany(v, ks...)

	return nil
}

func (l *Loader[K, V]) Wait() error {
	if err := l.closed(); err != nil {
		return err
	}

	close(l.done)

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

func (l *Loader[K, V]) load(v *V, k K) {
	l.mu.Lock()
	defer l.mu.Unlock()

	id := len(l.keys)
	l.keys = append(l.keys, k)
	l.kindByID[id] = one

	l.one[id] = v
}

func (l *Loader[K, V]) loadMany(v *[]V, ks ...K) {
	// NOTE: should we support multiple keys?
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, k := range ks {
		id := len(l.keys)
		l.keys = append(l.keys, k)
		l.kindByID[id] = many
		l.many[id] = v
	}
}

func (l *Loader[K, V]) wait() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Don't trigger batchFn if there are no keys.
	if len(l.keys) == 0 {
		return nil
	}

	keys := sliceutil.Dedup(l.keys)
	vals, err := l.batchFn(keys...)
	if err != nil {
		return err
	}

	valsByKey := make(map[K][]V)
	for _, v := range vals {
		v := v

		k, err := l.keyFn(v)
		if err != nil {
			return err
		}

		valsByKey[k] = append(valsByKey[k], v)
	}

	cached := make(map[K]bool)
	for i, k := range l.keys {
		kind := l.kindByID[i]
		v, ok := valsByKey[k]

		switch kind {
		case one:
			if !ok {
				return fmt.Errorf("%w: key '%v'", ErrKeyNotFound, k)
			}
			if len(v) != 1 {
				return fmt.Errorf("%w: key '%v'", ErrMultipleValuesFound, k)
			}
		}

		if cached[k] {
			// If there are duplicate keys, clone the subsequent value.
			// This prevents sharing reference for the same value, which is a common
			// mistake.
			c, err := copystructure.Copy(v)
			if err != nil {
				return err
			}

			switch kind {
			case one:
				*l.one[i] = c.([]V)[0]
			case many:
				*l.many[i] = append(*l.many[i], (c.([]V))...)
			}
		} else {
			switch kind {
			case one:
				*l.one[i] = v[0]
			case many:
				*l.many[i] = append(*l.many[i], v...)
			}
			cached[k] = true
		}
	}

	return nil
}
