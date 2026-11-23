# promise

Typed asynchronous results with manual settlement, cancellation and
combinators (`Race`, `Any`, `All`, `AllSettled`).

**Use it for:** starting work and awaiting it from several places, bridging
callbacks to values, or waiting on groups of results.

**Don't use it for:** coalescing duplicate calls by key (use `dsync/singleflight`
across processes) or batching key lookups (use [`sync/dataloader`](../dataloader)).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/promise
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/promise"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := promise.New(ctx, func(ctx context.Context) (int, error) { return 1, nil })
	b := promise.Resolve(ctx, 2)

	vals, err := promise.All(a, b)
	fmt.Println(vals, err) // [1 2] <nil>

	d := promise.Deferred[string](ctx)
	d.Resolve("done") // a Deferred must be settled or aborted
	fmt.Println(d.Await()) // done <nil>
}
```

## API

```go
func New[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) *Promise[T] // runs fn in a goroutine
func Deferred[T any](ctx context.Context) *Promise[T]                                      // manually settled
func Resolve[T any](ctx context.Context, v T) *Promise[T]
func Reject[T any](ctx context.Context, err error) *Promise[T]
func WithResolvers[T any](ctx context.Context) (p *Promise[T], resolve func(T), reject func(error))

func (*Promise[T]) Resolve(v T)
func (*Promise[T]) Reject(err error)
func (*Promise[T]) Abort(cause error) // cancel if unsettled; Await returns cause
func (*Promise[T]) Await() (T, error)
func (*Promise[T]) Status() Status    // StatusPending, StatusFulfilled, StatusRejected

func Race[T any](ps ...*Promise[T]) (T, error)
func Any[T any](ps ...*Promise[T]) (T, error)
func All[T any](ps ...*Promise[T]) ([]T, error)
func AllSettled[T any](ps ...*Promise[T]) []*Result[T]

type Map[K comparable, V any] = cache.Cache[K, Promise[V]] // weak cache of promises
func NewMap[K comparable, V any](ctx context.Context) *Map[K, V]

func NewChannel[T any](ctx context.Context) *Channel[T] // single-value channel
func (*Channel[T]) Send(v T)
func (*Channel[T]) Recv() (T, error)
func (*Channel[T]) Close(cause error)
func (*Channel[T]) Done() bool
```

## Behavior and guarantees

- **Single settlement.** The first of Resolve, Reject, Abort or the handler's
  return wins. `Await` can be called repeatedly and concurrently; all callers see
  the same result. A nil error passed to `Reject` produces a fulfilled zero
  result.
- **Settle or abort every promise (leak).** Each promise owns a child context
  registered with the parent. It is released only when the promise is settled or
  aborted. This applies to every promise from `Deferred` and to every promise
  handed out by `Map` (`NewMap` creates them with `Deferred`): the caller whose
  `LoadOrCreate` reports `loaded == false` owns settlement. Unsettled promises
  leak memory for as long as the parent context lives.
- **Cancellation is cooperative.** `Abort` or parent cancellation cancels the
  context given to `New`'s handler but does not wait for it; a handler that
  ignores its context keeps running.
- **Empty input panics.** `Race`, `Any`, `All` and `AllSettled` panic when called
  with no promises.
- **Combinators.** `Race` returns the first observed settlement, `Any` the first
  success or joined errors, `All` ordered values or the first observed error.
  `AllSettled` waits for all inputs (aborted ones count as rejected). `Race`,
  `Any`, `All` join their own waiters before returning and never abort inputs. If
  several inputs are already settled, which is observed first is unspecified.
- **`Channel.Done`.** Reports that a value was published, not that the channel
  was canceled or received from. `Recv` may still return the parent's
  cancellation cause in a narrow race with parent cancel, even when `Done` is
  true.
- **`Map`.** Built on [`sync/cache`](../cache): weakly held, so entries can be
  re-created after GC, and competing `Deferred` promises may be allocated per
  key with only one installed. There is no delete.
- Returned values are shared between awaiters; mutable values need caller
  synchronization.

## Testing

```sh
go vet ./... && go test -race -count=1 ./...
```
