# vector

Compute dot products, magnitudes, and cosine similarity.

## Installation

```sh
go get github.com/alextanhongpin/core/types/vector
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/vector"
)

func main() {
	a := vector.Vector{1, 0}
	b := vector.Vector{0, 1}
	fmt.Println(a.Dot(b), a.Magnitude(), a.CosineSimilarity(b))
}
```

## Behavior and limits

Dot panics for unequal lengths. CosineSimilarity returns zero when either magnitude is zero; otherwise it uses Dot and requires equal lengths. Vector is a mutable []float64 and does not normalize or copy its inputs.
