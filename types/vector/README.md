# vector

Dot product, Euclidean magnitude, and cosine similarity for dense `float64`
vectors.

**Use it for:** comparing small embedding or feature vectors without a numeric
library.

**Don't use it for:** large-scale nearest-neighbor search or anything that
needs SIMD, sparse vectors, or `float32` storage.

## Installation

```sh
go get github.com/alextanhongpin/core/types/vector
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/vector"
)

func main() {
	a := vector.Vector{1, 0}
	b := vector.Vector{1, 1}
	fmt.Printf("%.4f\n", a.CosineSimilarity(b))
}
```

It prints `0.7071`.

## API

```go
type Vector []float64

func (v Vector) Dot(o Vector) float64
func (v Vector) Magnitude() float64 // L2 norm
func (v Vector) CosineSimilarity(o Vector) float64
```

## Behavior and guarantees

- `Dot` and `CosineSimilarity` panic if the lengths differ.
- `CosineSimilarity` is in `[-1, 1]` (up to rounding) and returns 0 if either
  vector has zero magnitude.
- Values are not modified; methods are safe for concurrent use on a vector
  that is not being written.

## Testing

```sh
go vet ./vector/
go test -race -count=1 ./vector/
```

Run from the `types` directory.
