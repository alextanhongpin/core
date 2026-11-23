# sets

A generic set of comparable values with union, intersection, difference, and
subset tests, an order-preserving `Unique`, and a mutex-guarded `Sync` variant.

**Use it for:** membership checks and set algebra over IDs, tags, permissions,
and similar values.

**Don't use it for:** ordered collections (a `Set` has no order; use `Sorted`
when you need one) or values that are not comparable. For removing adjacent
duplicates from sorted input, `slices.Compact` avoids the map.

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

All `types` packages ship in the one `github.com/alextanhongpin/core/types`
module. Import this one as `github.com/alextanhongpin/core/types/sets`. Check
the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/sets"
)

func main() {
	fmt.Println(sets.Unique([]string{"b", "a", "b", "c", "a"}))

	admins := sets.Of("alice", "bob")
	active := sets.Of("bob", "carol")
	fmt.Println(sets.Sorted(sets.Intersection(admins, active)))
	fmt.Println(sets.Sorted(sets.Union(admins, active)))
	fmt.Println(sets.IsSubset(sets.Of("bob"), admins))
}
```

It prints:

```
[b a c]
[bob]
[alice bob carol]
true
```

## API

```go
func Unique[T comparable](vals []T) []T

type Set[T comparable] struct{ /* unexported */ }
func New[T comparable]() *Set[T]
func From[T comparable](s []T) *Set[T]
func Of[T comparable](vs ...T) *Set[T]
func (s *Set[T]) Add(v T) bool
func (s *Set[T]) AddMany(vs ...T) int
func (s *Set[T]) Remove(v T) bool
func (s *Set[T]) RemoveMany(vs ...T) int
func (s *Set[T]) Clear()
func (s *Set[T]) Len() int
func (s *Set[T]) IsEmpty() bool
func (s *Set[T]) Has(v T) bool
func (s *Set[T]) All() []T
func (s *Set[T]) Values() iter.Seq[T]
func (s *Set[T]) Filter(keep func(T) bool) *Set[T]
func (s *Set[T]) Clone() *Set[T]
func (s *Set[T]) String() string
func Sorted[T cmp.Ordered](s *Set[T]) []T

func Union[T comparable](a *Set[T], bs ...*Set[T]) *Set[T]
func Intersection[T comparable](a *Set[T], bs ...*Set[T]) *Set[T]
func Difference[T comparable](a *Set[T], bs ...*Set[T]) *Set[T]
func SymmetricDifference[T comparable](a, b *Set[T]) *Set[T]
func Equal[T comparable](a, b *Set[T]) bool
func IsSubset[T comparable](a, b *Set[T]) bool
func IsSuperset[T comparable](a, b *Set[T]) bool
func IsProperSubset[T comparable](a, b *Set[T]) bool
func IsProperSuperset[T comparable](a, b *Set[T]) bool
func IsDisjoint[T comparable](a, b *Set[T]) bool

type Sync[T comparable] struct{ /* unexported */ }
func NewSync[T comparable](s *Set[T]) *Sync[T]
// Add, AddMany, Remove, RemoveMany, Clear, Has, Len, IsEmpty, All, String
func (s *Sync[T]) Clone() *Set[T]
```

## Behavior and guarantees

- `Unique` keeps the first occurrence of each value, in input order. It returns
  nil for nil input.
- The zero values of `Set` and `Sync` are empty sets ready to use.
- `Add`/`Remove` report whether the set changed; `AddMany`/`RemoveMany` return
  how many values changed it.
- `All`, `Values`, and `String` use an unspecified order that can differ
  between calls. `Sorted` returns ascending order.
- `Union`, `Intersection`, `Difference`, `SymmetricDifference`, `Filter`, and
  `Clone` never modify their arguments and always return a new set, even when
  given a single set.
- Modifying a set while iterating `Values` is not allowed, except deleting the
  current value.
- A `Set` is not safe for concurrent use. `Sync` guards every method with a
  mutex. `NewSync` copies its argument (nil is allowed), and `Sync.Clone`
  returns an unsynchronized snapshot to pass to the set operations.

## Testing

```sh
go vet ./sets/...
go test -race -count=1 ./sets/...
```

Run from the `types` directory.
