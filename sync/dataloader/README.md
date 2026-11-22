# dataloader

Batch concurrent key lookups into fewer calls and share in-flight results
between callers that ask for the same key.

**Use it for:** GraphQL resolvers or request fan-out that would otherwise run one
database query per key.

**Don't use it for:** durable caching (results are weakly held), or
parallel batch execution (batches run serially). For general promise
composition see [`sync/promise`](../promise).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/dataloader
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/dataloader"
)

func main() {
	ctx := context.Background()
	fetch := func(ctx context.Context, ids []int) (map[int]string, error) {
		out := make(map[int]string, len(ids))
		for _, id := range ids {
			out[id] = fmt.Sprintf("user-%d", id)
		}
		return out, nil
	}
	dl, stop, err := dataloader.New(ctx, fetch, dataloader.Config{})
	if err != nil {
		panic(err)
	}
	defer stop()

	name, err := dl.LoadContext(ctx, 7)
	fmt.Println(name, err) // user-7 <nil>
}
```

## API

```go
func New[K comparable, V any](ctx context.Context, fn func(ctx context.Context, keys []K) (map[K]V, error), cfg Config) (*DataLoader[K, V], func(), error)
func MustNew[K comparable, V any](ctx context.Context, fn func(context.Context, []K) (map[K]V, error), cfg Config) (*DataLoader[K, V], func()) // panics on invalid cfg

func (*DataLoader[K, V]) Load(key K) (V, error)                            // waits using the loader context
func (*DataLoader[K, V]) LoadContext(ctx context.Context, key K) (V, error) // ctx cancels only this wait
func (*DataLoader[K, V]) LoadMany(keys ...K) ([]*Result[K, V], error)      // results in input order

func Func[K comparable, V, T any](fn func(context.Context, V) (T, error), dl interface {
	LoadContext(context.Context, K) (V, error)
}) func(context.Context, K) (T, error) // load a V by key, then transform it

type Config struct{ BatchInterval time.Duration; BatchSize, BufferSize int }
func DefaultConfig() Config // 16ms, 25 keys, unbuffered
func (Config) WithDefaults() Config
func (Config) Validate() error

var ErrNotFound, ErrCanceled error
```

## Behavior and guarantees

- **Batching.** Keys are collected until `BatchSize` or `BatchInterval`, then the
  batch function is called once. Zero `BatchSize`/`BatchInterval` select 25 and
  16ms; negative values and a nil batch function make `New` return an error.
- **Batches run serially.** One worker calls the batch function; a slow batch
  delays every other key queued behind it, including unrelated keys. Keep the
  batch function fast or bounded by its context.
- **Missing keys.** Keys absent from the returned map fail with `ErrNotFound`.
  A batch error rejects every key in that batch.
- **Sharing.** Concurrent callers for one key share a future. Results are held
  weakly ([`sync/cache`](../cache)) and may be loaded again after GC; this is
  not a result cache. Returned mutable values are shared.
- **Cancellation.** `LoadContext` cancellation stops only that caller's wait;
  the shared load continues for others. If it races with completion, either
  outcome may be returned. `Load` and `LoadMany` wait on the loader lifetime
  context.
- **Admission.** `LoadContext` admits new keys through loader-owned goroutines,
  so `BufferSize` does not bound pending keys; bound caller concurrency when key
  cardinality is unbounded.
- **Stop.** The returned stop function is idempotent and concurrent-safe. It
  cancels pending loads with `ErrCanceled` and waits for the worker. The batch
  function must honor its context and must not call stop. Loads after stop return
  the cancellation cause.

Use `errors.Is` to check `ErrNotFound`, `ErrCanceled` and context errors.

## Testing

The test suite depends on the private `evaltest` module.

```sh
go vet ./... && go test -race -count=1 ./...
```
