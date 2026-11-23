package sets

import "sync"

// Sync is a Set guarded by a mutex, safe for concurrent use. The zero value
// is an empty set ready to use. Set operations such as Union take a *Set;
// use Clone to get a snapshot.
type Sync[T comparable] struct {
	mu  sync.Mutex
	set Set[T]
}

// NewSync returns a concurrent set holding the values of s. s is copied, so
// later changes to s do not affect the result. s may be nil.
func NewSync[T comparable](s *Set[T]) *Sync[T] {
	out := &Sync[T]{}
	if s != nil {
		out.set = *s.Clone()
	}
	return out
}

// Add inserts v and reports whether it was not already present.
func (s *Sync[T]) Add(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Add(v)
}

// AddMany inserts vs and returns how many were not already present.
func (s *Sync[T]) AddMany(vs ...T) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.AddMany(vs...)
}

// Remove deletes v and reports whether it was present.
func (s *Sync[T]) Remove(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Remove(v)
}

// RemoveMany deletes vs and returns how many were present.
func (s *Sync[T]) RemoveMany(vs ...T) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.RemoveMany(vs...)
}

// Clear removes every value.
func (s *Sync[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Clear()
}

// Has reports whether v is in the set.
func (s *Sync[T]) Has(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Has(v)
}

// Len returns the number of values.
func (s *Sync[T]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Len()
}

// IsEmpty reports whether the set has no values.
func (s *Sync[T]) IsEmpty() bool {
	return s.Len() == 0
}

// All returns the values as a new slice in unspecified order.
func (s *Sync[T]) All() []T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.All()
}

// Clone returns an unsynchronized copy of the current values.
func (s *Sync[T]) Clone() *Set[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.Clone()
}

// String formats the set as "{a, b, c}" in unspecified order.
func (s *Sync[T]) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.set.String()
}
