# queue

Maintain a generic priority queue with optional bounded retention.

## Installation

```sh
go get github.com/alextanhongpin/core/types/queue
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"fmt"
	"github.com/alextanhongpin/core/types/queue"
)

type item int

func (a item) Less(b item) bool { return a < b }

func main() {
	q := queue.NewPriorityQueue[item]()
	q.TopK = 2
	q.Push(1, 5, 3)
	value, ok := q.Pop()
	fmt.Println(value, ok) // 3 true
}
```

## Behavior and limits

Items implement Less(T) bool. Pop returns the least item under Less, with ok=false when empty. A positive TopK removes the least items after insertion, retaining the greatest K. Zero or negative TopK leaves the queue unbounded. Slice returns a sorted copy. Synchronize shared access externally.
