# reservoir

Accumulate a uniform reservoir sample from a stream of unknown size.

## Installation

```sh
go get github.com/alextanhongpin/core/types/reservoir
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"github.com/alextanhongpin/core/types/reservoir"
)

func main() {
	sample := reservoir.NewReservoirSampling[int](10)
	for i := range 100 {
		sample.Add(i)
	}
}
```

## Behavior and limits

Use a nonnegative capacity; a negative capacity panics during allocation. Add keeps at most k stored items using math/rand/v2. The current public API has no accessor for the stored sample, so callers cannot retrieve it. Instances do not synchronize concurrent access.
