# list

Generic slice helpers that the standard library's `slices` and `maps` packages
do not provide: map, filter, reduce, group, partition, and key-based
deduplication. Every function takes and returns plain slices and never modifies
its input.

**Use it for:** short transformations over slices, such as turning `[]User` into
`[]string`, or grouping records by a field.

**Don't use it for:** anything the standard library already does. Use
`slices.ContainsFunc` (any/all/none), `slices.IndexFunc` (find), `slices.Reverse`,
`slices.Chunk`, `slices.Concat` (flatten), `slices.Max`/`slices.Min`, and plain
slicing (`s[:n]`, `s[n:]`). For distinct comparable values use
[`sets.Unique`](../sets).

## Installation

```sh
go get github.com/alextanhongpin/core/types
```

All `types` packages ship in the one `github.com/alextanhongpin/core/types`
module. Import this one as `github.com/alextanhongpin/core/types/list`. Check
the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"
	"strconv"

	"github.com/alextanhongpin/core/types/list"
)

func main() {
	ids, err := list.MapError([]string{"3", "1", "4"}, strconv.Atoi)
	if err != nil {
		panic(err)
	}
	even, odd := list.Partition(ids, func(n int) bool { return n%2 == 0 })
	byLen := list.GroupBy([]string{"go", "rust", "c", "zig"}, func(s string) int { return len(s) })

	fmt.Println(list.Sum(ids), even, odd, byLen[2])
}
```

It prints `8 [4] [3 1] [go]`.

## API

```go
func Map[T, U any](s []T, fn func(T) U) []U
func MapError[T, U any](s []T, fn func(T) (U, error)) ([]U, error)
func Filter[T any](s []T, keep func(T) bool) []T
func FlatMap[T, U any](s []T, fn func(T) []U) []U
func Reduce[T, U any](s []T, initial U, fn func(acc U, v T) U) U
func GroupBy[T any, K comparable](s []T, key func(T) K) map[K][]T
func Partition[T any](s []T, pred func(T) bool) (matched, rest []T)
func UniqueFunc[T any, K comparable](s []T, key func(T) K) []T
func Sum[T constraints.Integer | constraints.Float](s []T) T
```

## Behavior and guarantees

- Inputs are never modified; results are new slices that do not alias the input.
- Order is preserved: `Map`, `Filter`, `FlatMap`, `Partition`, `GroupBy` (within
  each group), and `UniqueFunc` (first element per key wins).
- `Map`, `MapError`, and `UniqueFunc` return nil for a nil input and an empty
  slice for an empty input. `Filter`, `FlatMap`, and `Partition` return nil when
  nothing is collected.
- `MapError` stops at the first error, returns it unwrapped with a nil slice, and
  does not call `fn` for later elements.
- `Sum` of an empty slice is zero; integer sums wrap on overflow.
- Functions are as safe for concurrent use as the callbacks you pass.

## Testing

```sh
go vet ./list/...
go test -race -count=1 ./list/...
```

Run from the `types` directory.
