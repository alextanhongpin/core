package list

import (
	"iter"
	"slices"
)

// Iter is a generic iterator for type T.
type Iter[T any] struct {
	iter iter.Seq[T] // The underlying sequence function
}

// IterOf creates an iterator from a variadic list of values.
func IterOf[T any](vs ...T) *Iter[T] {
	return IterFrom(vs)
}

// IterFrom creates an iterator from a slice of values.
func IterFrom[T any](vs []T) *Iter[T] {
	return &Iter[T]{
		iter: slices.Values(vs),
	}
}

func (l *Iter[T]) Iter() iter.Seq[T] {
	return l.iter
}

// Filter filters elements based on the provided predicate function.
func (l *Iter[T]) Filter(fn func(T) bool) *Iter[T] {
	old := l.iter
	l.iter = func(yield func(T) bool) {
		for v := range old {
			if fn(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
	return l
}

// IterMap creates a new iterator by applying the function to each element of
// the input iterator.
func IterMap[T, V any](iterator Iter[T], fn func(T) V) *Iter[V] {
	return &Iter[V]{
		iter: func(yield func(V) bool) {
			for v := range iterator.Iter() {
				if !yield(fn(v)) {
					return
				}
			}
		},
	}
}

// Map transforms each element using the provided function.
func (l *Iter[T]) Map(fn func(T) T) *Iter[T] {
	old := l.iter
	l.iter = func(yield func(T) bool) {
		for v := range old {
			if !yield(fn(v)) {
				return
			}
		}
	}
	return l
}

// Each applies the function to each element in the iterator.
func (l *Iter[T]) Each(fn func(T)) {
	for v := range l.iter {
		fn(v)
	}
}

// Collect gathers all elements from the iterator into a slice.
func (l *Iter[T]) Collect() []T {
	return slices.Collect(l.iter)
}

// Reverse returns a new iterator with the elements in reverse order.
func (i *Iter[T]) Reverse() *Iter[T] {
	collect := i.Collect()
	counter := len(collect) - 1
	for e := range i.iter {
		collect[counter] = e
		counter--
	}
	return IterFrom(collect)
}

// List converts the iterator to a List.
func (i *Iter[T]) List() *List[T] {
	return From(i.Collect())
}

// Take limits the iterator to at most n elements.
func (l *Iter[T]) Take(n int) *Iter[T] {
	old := l.iter
	l.iter = func(yield func(T) bool) {
		count := 0
		for v := range old {
			if count >= n {
				return
			}
			if !yield(v) {
				return
			}
			count++
		}
	}
	return l
}

// Skip skips the first n elements in the iterator.
func (l *Iter[T]) Skip(n int) *Iter[T] {
	old := l.iter
	l.iter = func(yield func(T) bool) {
		count := 0
		for v := range old {
			if count < n {
				count++
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
	return l
}

// Any returns true if any element matches the predicate function.
func (l *Iter[T]) Any(fn func(T) bool) bool {
	for v := range l.iter {
		if fn(v) {
			return true
		}
	}
	return false
}

// All returns true if all elements match the predicate function.
func (l *Iter[T]) All(fn func(T) bool) bool {
	for v := range l.iter {
		if !fn(v) {
			return false
		}
	}
	return true
}

// Find returns the first element matching the predicate function, or false if not found.
func (l *Iter[T]) Find(fn func(T) bool) (T, bool) {
	for v := range l.iter {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Count returns the number of elements in the iterator.
func (l *Iter[T]) Count() int {
	count := 0
	for range l.iter {
		count++
	}
	return count
}

// Reduce reduces the elements to a single value using the provided function and initial value.
func (l *Iter[T]) Reduce(fn func(acc, v T) T, initial T) T {
	acc := initial
	for v := range l.iter {
		acc = fn(acc, v)
	}
	return acc
}

type Deduplicator[T any] interface {
	Has(T) bool
	Set(T)
}

// DedupFunc removes duplicates using a key function.
func (l *Iter[T]) DedupFunc(fn Deduplicator[T]) *Iter[T] {
	old := l.iter
	l.iter = func(yield func(T) bool) {
		for v := range old {
			if !fn.Has(v) {
				fn.Set(v)
				if !yield(v) {
					return
				}
			}
		}
	}

	return l
}
