# queue

A generic binary-heap priority queue, with an optional bound that keeps only
the highest `TopK` items.

**Use it for:** scheduling by priority, merging sorted streams, or keeping a
running top-k leaderboard.

**Don't use it for:** FIFO queues (use a slice or channel), concurrent
producers and consumers (add your own locking), or updating an item's priority
in place.

## Installation

```sh
go get github.com/alextanhongpin/core/types/queue
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/types/queue"
)

type Task struct {
	Name     string
	Priority int
}

func (t Task) Less(o Task) bool { return t.Priority < o.Priority }

func main() {
	var pq queue.PriorityQueue[Task]
	pq.Push(Task{"write", 2}, Task{"deploy", 3}, Task{"lint", 1})
	for pq.Len() > 0 {
		t, _ := pq.Pop()
		fmt.Println(t.Name)
	}

	top := queue.PriorityQueue[Task]{TopK: 2}
	top.Push(Task{"a", 5}, Task{"b", 1}, Task{"c", 9}, Task{"d", 3})
	fmt.Println(top.Slice())
}
```

It prints `lint`, `write`, `deploy`, then `[{a 5} {c 9}]`.

## API

```go
type Orderable[T any] interface{ Less(other T) bool }

type PriorityQueue[T Orderable[T]] struct {
	TopK int // when positive, keep only the TopK highest items
}

func NewPriorityQueue[T Orderable[T]]() *PriorityQueue[T]

func (pq *PriorityQueue[T]) Push(vals ...T)
func (pq *PriorityQueue[T]) Pop() (T, bool)  // lowest item
func (pq *PriorityQueue[T]) Peek() (T, bool) // lowest item, not removed
func (pq *PriorityQueue[T]) Len() int
func (pq *PriorityQueue[T]) Slice() []T      // ascending copy
```

## Behavior and guarantees

- It is a min-heap: `Pop` and `Peek` return the item for which `Less` reports
  lowest. Reverse `Less` for a max-heap.
- With `TopK > 0`, each `Push` discards the lowest items until at most `TopK`
  remain. Lowering `TopK` takes effect on the next `Push`.
- `Less` must be a strict weak ordering. Order among equal items is not
  defined by `Pop`; `Slice` sorts stably from heap order, so it is not
  insertion order either.
- `Push` and `Pop` are O(log n); `Slice` is O(n log n) and does not modify the
  queue. `Pop` clears the vacated slot so popped values can be collected.
- The zero value is an empty, unbounded queue. Not safe for concurrent use.

## Testing

```sh
go vet ./queue/
go test -race -count=1 ./queue/
```

Run from the `types` directory.
