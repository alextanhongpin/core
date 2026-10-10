# topk

Select the values with the greatest float32 scores from an iterator.

## Installation

```sh
go get github.com/alextanhongpin/core/types/topk
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/topk"
)

func main() {
	seq := func(yield func(float32, string) bool) {
		if !yield(1, "low") {
			return
		}
		yield(5, "high")
	}
	values, err := topk.TopK[string](seq, 1)
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
}
```

## Behavior and limits

Pass a positive k: zero explicitly panics and negative values fail allocation. Results are sorted by descending score and contain at most k values. Equal scores do not replace retained items once full. Selection scans the retained items for each candidate, taking O(n*k) time and O(k) storage. The current implementation returns nil error on success.
