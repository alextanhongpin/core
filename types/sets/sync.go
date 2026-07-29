package sets

import (
	"cmp"
	"sync"
)

// Sync wraps an existing Set to provide thread-safe operations.
type Sync[T cmp.Ordered] struct {
	set *Set[T]
	mu  sync.Mutex
}

// NewSync creates a new thread-safe set wrapping the provided base set.
func NewSync[T cmp.Ordered](s *Set[T]) *Sync[T] {
	return &Sync[T]{
		set: s,
	}
}

// Add adds an element to the synchronized set safely.
func (s *Sync[T]) Add(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Add(v)
}

// AddMany adds multiple elements to the synchronized set safely.
func (s *Sync[T]) AddMany(vs ...T) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.AddMany(vs...)
}

// Remove removes an element from the synchronized set safely.
func (s *Sync[T]) Remove(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Remove(v)
}

// RemoveMany removes multiple elements from the synchronized set safely.
func (s *Sync[T]) RemoveMany(vs ...T) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.RemoveMany(vs...)
}

// Clear removes all elements from the synchronized set safely.
func (s *Sync[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Clear()
}

// Has checks if an element exists in the synchronized set safely.
func (s *Sync[T]) Has(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Has(v)
}

// Len returns the size of the synchronized set safely.
func (s *Sync[T]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Len()
}

// IsEmpty returns true if the synchronized set contains no elements.
func (s *Sync[T]) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.IsEmpty()
}

// All returns all elements of the synchronized set in consistent order safely.
func (s *Sync[T]) All() []T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.All()
}

// String returns a string representation of the synchronized set safely.
func (s *Sync[T]) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.String()
}

// Clone creates a thread-safe deep copy of the set.
func (s *Sync[T]) Clone() *Sync[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return NewSync(s.set.Clone())
}

// Range iterates over all elements in the synchronized set safely.
func (s *Sync[T]) Range(predicate func(T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Range(predicate)
}

// Filter returns a new synchronized set containing only elements that satisfy the predicate.
func (s *Sync[T]) Filter(predicate func(T) bool) *Sync[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return NewSync(s.set.Filter(predicate))
}

// Any returns true if any element in the synchronized set satisfies the predicate safely.
func (s *Sync[T]) Any(predicate func(T) bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Any(predicate)
}

// Every returns true if all elements in the synchronized set satisfy the predicate safely.
func (s *Sync[T]) Every(predicate func(T) bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Every(predicate)
}
