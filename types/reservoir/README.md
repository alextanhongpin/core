# reservoir

Keep a uniform random sample of at most `k` items from a stream of unknown
length, in O(k) memory (Algorithm R).

**Use it for:** sampling log lines, requests, or rows when you cannot hold the
whole stream.

**Don't use it for:** weighted sampling, or sampling from a slice you already
have (use `math/rand/v2` `Perm` or `Shuffle`).

## Installation

```sh
go get github.com/alextanhongpin/core/types/reservoir
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/reservoir"
)

func main() {
	r := reservoir.NewReservoirSampling[int](3)
	for i := range 1000 {
		r.Add(i)
	}
	fmt.Println(len(r.Sample()), r.Count())
}
```

It prints `3 1000`; the three sampled values vary per run.

## API

```go
func NewReservoirSampling[T any](k int) *ReservoirSampling[T] // panics if k <= 0

func (r *ReservoirSampling[T]) Add(v T)
func (r *ReservoirSampling[T]) Sample() []T // copy, min(k, Count()) items
func (r *ReservoirSampling[T]) Count() int  // items offered so far
```

## Behavior and guarantees

- After `n` calls to `Add`, each item seen is in the sample with probability
  `min(k, n)/n`.
- Until `k` items have been added, `Sample` returns them in insertion order;
  after that the order carries no meaning.
- Randomness comes from `math/rand/v2`'s global source and cannot be seeded.
- Not safe for concurrent use.

## Testing

```sh
go vet ./reservoir/
go test -race -count=1 ./reservoir/
```

Run from the `types` directory.
