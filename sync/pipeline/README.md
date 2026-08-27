# Pipeline

A lightweight, generic Go library for building concurrent data-processing pipelines with channels.

The package provides composable stages for sources, transformations, flow control, batching, fan-in/out, deduplication, rate limiting and sinks. All APIs are generic and work with any type.

## Installation

```bash
go get github.com/alextanhongpin/core/sync/pipeline
```

Requires Go 1.21+.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func main() {
	ctx := context.Background()

	// Source from a slice
	src := pipeline.SourceSlice(ctx, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	// Transform with 3 workers, keep only even numbers
	evens := pipeline.PipeN(src, func(n int) (int, bool) {
		if n%2 == 0 {
			return n * 2, true
		}
		return 0, false
	}, 3)

	// Collect results
	for v := range evens {
		fmt.Println(v)
	}
}
```

## Sources

Create channels from existing data with context cancellation.

```go
// From a slice
ch := pipeline.SourceSlice(ctx, []string{"a", "b", "c"})

// From an existing channel, with context cancellation
ch := pipeline.SourceChan(ctx, in)

// From an iter.Seq
ch := pipeline.SourceIter(ctx, func(yield func(int)) { for i:=0; i<5; i++ { yield(i) } })
```

## Stages

### Pipe / PipeN

`Pipe` transforms items, `PipeN` runs with `n` workers.

```go
out := pipeline.Pipe(in, func(v string) (int, bool) {
    n, err := strconv.Atoi(v)
    if err { return 0, false }
    return n, true
})

out := pipeline.PipeN(in, fn, 4)
```

### Batch

Collect items into slices by size or timeout.

```go
batches := pipeline.Batch(in, 100, 1*time.Second)
for batch := range batches {
    // len(batch) <= 100
}
```

### Buffer

Add a buffered stage.

```go
buf := pipeline.Buffer(in, 128)
```

### Debounce

Emit at most one item per `duration`.

```go
debounced := pipeline.Debounce(in, 200*time.Millisecond)
```

### Dedup / DedupFunc

Remove duplicates.

```go
uniq := pipeline.Dedup(in)               // comparable
uniq := pipeline.DedupFunc(in, func(v MyType) string { return v.ID })
```

### FanIn / FanOut

Merge multiple channels or distribute to N channels.

```go
merged := pipeline.FanIn(ch1, ch2, ch3)
outs := pipeline.FanOut(in, 3) // round-robin
```

### Merge

Merge channels with a custom combine function.

```go
merged := pipeline.Merge(func(a, b int) int { return a + b }, ch1, ch2)
```

### RateLimit

Limit throughput.

```go
limited := pipeline.RateLimit(in, 10, time.Second) // ~10 items per second
```

### Semaphore

Limit concurrent execution of a function.

```go
out := pipeline.Semaphore(in, func(v int) string {
    return fmt.Sprintf("%d", v)
}, 5)
```

### Tee

Split a stream into two identical streams.

```go
a, b := pipeline.Tee(in)
```

## Sinks

```go
// Collect to slice
vals := pipeline.Collect(in)

// Reduce with error handling
sum, err := pipeline.Reduce(in, func(v, acc int) (int, error) { return acc+v, nil }, 0)

// Consume without storing
pipeline.Sink(in, func(v T) { ... })

// Count items
n := pipeline.Count(in)
```

## Utilities

```go
pipeline.SafeClose(ch)
pipeline.Flush(in) // drain
```

## Examples

### Batch + workers

```go
src := pipeline.SourceSlice(ctx, data)
batched := pipeline.Batch(src, 50, 500*time.Millisecond)
out := pipeline.PipeN(batched, func(batch []T) (Result, bool) {
    // process batch
    return process(batch), true
}, 4)
```

### Fan-out / fan-in

```go
outs := pipeline.FanOut(in, 4)
for _, o := range outs {
    o := pipeline.PipeN(o, worker, 2)
    results = append(results, o)
}
merged := pipeline.FanIn(results...)
```

## License

MIT
