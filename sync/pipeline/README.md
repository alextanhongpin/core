# pipeline

Typed channel stages: sources, transformations, flow control and sinks that
compose into in-process pipelines.

**Use it for:** streaming values through a few goroutine-backed stages
(transform, batch, fan out, throttle) with plain channels.

**Don't use it for:** broadcasting to dynamic subscribers (use
[`sync/broadcast`](../broadcast)), collecting errors from stages (there is no
error channel), or work that needs per-item retries (see
[`sync/retry`](../retry)).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/pipeline
```

Requires Go 1.25+.

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/pipeline"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := pipeline.SourceSlice(ctx, []int{1, 2, 3})
	doubled := pipeline.PipeCtx(ctx, in, func(v int) (int, bool) { return v * 2, true })
	fmt.Println(pipeline.Collect(doubled))
}
```

It prints `[2 4 6]`.

## API

```go
// Sources (cancellable)
func SourceChan[T any](ctx context.Context, in <-chan T) <-chan T
func SourceIter[T any](ctx context.Context, seq iter.Seq[T]) <-chan T
func SourceSlice[T any](ctx context.Context, values []T) <-chan T

// Stages without ctx: consumers must drain the output until closed
func Batch[T any](in <-chan T, size int, timeout time.Duration) <-chan []T
func Pipe[T, V any](in <-chan T, fn func(T) (V, bool)) <-chan V
func PipeN[T, V any](in <-chan T, fn func(T) (V, bool), n int) <-chan V
func Semaphore[T, V any](in <-chan T, fn func(T) V, n int) <-chan V
func Tee[T any](in <-chan T) (<-chan T, <-chan T)

// Cancellable variants of the above
func BatchCtx[T any](ctx context.Context, in <-chan T, size int, timeout time.Duration) <-chan []T
func PipeCtx[T, V any](ctx context.Context, in <-chan T, fn func(T) (V, bool)) <-chan V
func PipeNCtx[T, V any](ctx context.Context, in <-chan T, fn func(T) (V, bool), n int) <-chan V
func SemaphoreCtx[T, V any](ctx context.Context, in <-chan T, fn func(T) V, n int) <-chan V
func TeeCtx[T any](ctx context.Context, in <-chan T) (<-chan T, <-chan T)

// Other stages
func Buffer[T any](in <-chan T, cap int) <-chan T
func Debounce[T any](in <-chan T, duration time.Duration) <-chan T // leading-edge throttle
func Dedup[T comparable](in <-chan T) <-chan T
func DedupFunc[T any, V comparable](in <-chan T, fn func(T) V) <-chan T
func FanIn[T any](channels ...<-chan T) <-chan T
func FanOut[T any](in <-chan T, n int) []<-chan T
func Merge[T any](mergeFn func(T, T) T, channels ...<-chan T) <-chan T
func RateLimit[T any](in <-chan T, every int, interval time.Duration) <-chan T
func SafeClose[T any](ch chan T) (closed bool)

// Sinks (drain until closed)
func Collect[T any](in <-chan T) []T
func Sink[T any](in <-chan T, fn func(T))
func Flush[T any](in <-chan T)
func Count[T any](in <-chan T) int
func Reduce[T, V any](in <-chan T, fn func(T, V) (V, error), start V) (V, error)
```

## Behavior and guarantees

- **Draining:** every stage runs a goroutine that closes its output when input
  ends. Stages without a context (including `Buffer`, `Debounce`, `Dedup`,
  `DedupFunc`, `FanIn`, `FanOut`, `Merge`, `RateLimit`) block on send, so
  consumers must drain every output until closed, or the goroutine leaks.
- **Cancellation:** `SourceChan`, `SourceIter`, `SourceSlice` and the `*Ctx`
  stages stop reading, stop sending, close their outputs and exit when `ctx` is
  canceled, even if no one is receiving. `BatchCtx` discards a partial batch on
  cancel. `PipeNCtx` and `SemaphoreCtx` wait for callbacks already running
  before closing the output. `SourceIter` cannot interrupt an iterator blocked
  before it yields.
- **Ordering:** `Pipe`, `Batch`, `Buffer`, `Dedup`, `Debounce` preserve order.
  `PipeN` and `Semaphore` bound concurrent callbacks and may reorder results.
  `FanIn` keeps each input's order with unspecified interleaving.
- **Head-of-line blocking:** `FanOut` (round-robin, one output at a time) and
  `Tee` (both outputs must receive each value) stall on a slow or undrained
  output.
- **Merge:** `mergeFn` is applied only opportunistically, when both inputs
  happen to be ready; pairing and output timing are nondeterministic.
- **Dedup / DedupFunc:** keep every seen key for the stage's lifetime; memory is
  unbounded for unbounded distinct keys.
- **Debounce:** a leading-edge throttle; it emits an item and drops all items
  within the following duration. It is not trailing-edge debounce.
- **Batch:** flushes at `size`, `timeout` after the first item, or when input
  closes. `size` must be positive.
- **Reduce:** returns on the first callback error; the caller must cancel the
  source and `Flush` the input so upstream stages can exit.
- **Errors/panics:** no stage has an error channel or recovers callback panics.
  `SafeClose` only recovers a close panic; it does not make concurrent send and
  close safe.
- `SourceSlice` reads the caller's slice without copying; do not mutate it
  while the source runs. Callbacks and mutable payloads need caller
  synchronization.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
go test -count=1 ./...
```
