# DataLoader

A Go implementation of the DataLoader pattern for efficient batch loading of data. Solves the N+1 query problem by grouping concurrent individual requests into batches, with promise-based deduplication.

## Features

- **Batch Loading**: Groups multiple concurrent requests into batches to reduce database/API calls
- **Promise-Based Deduplication**: Concurrent loads for the same key share a single in-flight request
- **Configurable Batching**: Tune batch size, interval, and channel buffer to match your workload
- **Concurrency Safe**: Thread-safe operations for concurrent access
- **Graceful Shutdown**: Stop function waits for in-flight work and cancels pending promises
- **Generic Support**: Full Go generics support for type safety

## Installation

```bash
go get github.com/alextanhongpin/core/sync/dataloader
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/alextanhongpin/core/sync/dataloader"
)

func main() {
    ctx := context.Background()

    dl, stop := dataloader.New(ctx, func(ctx context.Context, keys []string) (map[string]int, error) {
        m := make(map[string]int)
        for _, k := range keys {
            n, err := strconv.Atoi(k)
            if err != nil {
                continue
            }
            m[k] = n
        }
        return m, nil
    }, dataloader.Config{
        BatchInterval: 16 * time.Millisecond,
        BatchSize:     25,
    })
    defer stop()

    v, err := dl.Load("42")
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println("value:", v) // 42
}
```

## API Reference

### `Config`

```go
type Config struct {
    BatchInterval time.Duration // How long to wait before dispatching a batch (default: 16ms)
    BatchSize     int           // Maximum number of keys per batch (default: 25)
    BufferSize    int           // Channel buffer size for incoming keys (default: 0)
}
```

Pass `Config` by value or pointer — both are accepted. Pass `nil` to use defaults:

```go
// by value
dl, stop := dataloader.New(ctx, batchFn, dataloader.Config{BatchSize: 50})

// by pointer
dl, stop := dataloader.New(ctx, batchFn, &dataloader.Config{BatchSize: 50})

// nil → uses DefaultConfig (BatchInterval: 16ms, BatchSize: 25)
dl, stop := dataloader.New(ctx, batchFn, nil)
```

### `New`

```go
func New[K comparable, V any](
    ctx context.Context,
    fn func(ctx context.Context, keys []K) (map[K]V, error),
    cfg *Config,
) (*DataLoader[K, V], func())
```

Returns the dataloader and a **stop function**. Call `stop()` (e.g. via `defer`) to cancel the internal context, wait for the background worker to finish, and reject any pending promises with `ErrCanceled`.

### `Result`

```go
type Result[K comparable, V any] struct {
    Key   K
    Value V
    Error error
}
```

### Methods

#### `Load(key K) (V, error)`

Loads a single value by key. Concurrent callers with the same key share one in-flight request. Returns `ErrNotFound` if the batch function did not include the key in its result map.

#### `LoadMany(keys ...K) ([]*Result[K, V], error)`

Loads multiple values. Returns one `*Result` per key in input order. The outer error is always `nil`; per-key errors are in `Result.Error`.

#### `Delete(key K, err error)`

Removes a key from the in-flight promise map and rejects its promise with the given error. Blocks until the promise is settled.

### Sentinel Errors

| Error | Description |
|---|---|
| `ErrNotFound` | The batch function returned no value for a key |
| `ErrCanceled` | The dataloader was stopped before the key could be loaded |

### `Func` Helper

`Func` composes a dataloader with a downstream transformation function, returning a single function that loads by key and then applies `fn` to the result.

```go
func Func[K comparable, V, T any](
    fn func(ctx context.Context, req V) (T, error),
    dl dataloader[K, V],
) func(ctx context.Context, k K) (T, error)
```

**Example:**

```go
type User struct {
    ID   int
    Name string
}

dl, stop := dataloader.New(ctx, fetchUsers, nil)
defer stop()

greet := dataloader.Func(func(ctx context.Context, u User) (string, error) {
    return fmt.Sprintf("hi, %s", u.Name), nil
}, dl)

msg, err := greet(ctx, "alice") // loads User for "alice", then greets
```

## Examples

### Concurrent Loads with Batching

Multiple goroutines loading concurrently will have their keys collected and dispatched in a single batch:

```go
dl, stop := dataloader.New(ctx, batchFn, &dataloader.Config{
    BatchInterval: 16 * time.Millisecond,
    BatchSize:     25,
})
defer stop()

var wg sync.WaitGroup
results := make([]int, 5)
keys := []string{"1", "2", "3", "4", "5"}

for i, key := range keys {
    wg.Go(func() {
        v, err := dl.Load(key)
        if err == nil {
            results[i] = v
        }
    })
}
wg.Wait()
// batchFn is called once with up to all 5 keys
```

### Loading Multiple Keys at Once

```go
results, _ := dl.LoadMany("1", "2", "999")

for _, r := range results {
    switch {
    case errors.Is(r.Error, dataloader.ErrNotFound):
        fmt.Printf("key %q not found\n", r.Key)
    case r.Error != nil:
        fmt.Printf("key %q error: %v\n", r.Key, r.Error)
    default:
        fmt.Printf("key %q = %d\n", r.Key, r.Value)
    }
}
```

### Handling Cancellation

```go
dl, stop := dataloader.New(ctx, batchFn, nil)
stop() // stop before any loads

_, err := dl.Load("1")
// errors.Is(err, dataloader.ErrCanceled) == true
```

### GraphQL / N+1 Resolution

Create one loader per request, use concurrent goroutines to load — all hits will be batched:

```go
userLoader, stop := dataloader.New(ctx, func(ctx context.Context, ids []string) (map[string]User, error) {
    return db.QueryUsers(ctx, ids)
}, nil)
defer stop()

var wg sync.WaitGroup
for i := range posts {
    wg.Go(func() {
        u, err := userLoader.Load(posts[i].AuthorID)
        if err == nil {
            posts[i].Author = &u
        }
    })
}
wg.Wait()
```

## Error Handling

Errors can occur at two levels:

**Batch-level** — if `batchFn` returns a non-nil error, *all keys* in that batch receive the error:

```go
v, err := dl.Load("key")
if err != nil && !errors.Is(err, dataloader.ErrNotFound) {
    // batch-level failure
}
```

**Key-level** — if `batchFn` succeeds but omits a key from the result map, that key gets `ErrNotFound`:

```go
v, err := dl.Load("missing")
if errors.Is(err, dataloader.ErrNotFound) {
    // key was not returned by the batch function
}
```

## Performance Considerations

| Parameter | Effect |
|---|---|
| `BatchInterval` | Lower → less latency, smaller batches. Higher → larger batches, more efficient DB calls. |
| `BatchSize` | Cap on keys per batch. Prevents unbounded memory usage per dispatch. |
| `BufferSize` | Non-zero lets producers enqueue keys without blocking when the dispatcher is busy. |

- **16 ms** is a sensible default `BatchInterval` for typical web request lifecycles.
- Tune `BatchSize` to match your backend's optimal query size (e.g. SQL `IN` clause limits).
- `BufferSize = 0` (default) means producers block until the background worker picks up the key.

## Testing

```go
func TestDataloader_ErrNotFound(t *testing.T) {
    dl, stop := dataloader.New(ctx,
        func(ctx context.Context, keys []string) (map[string]int, error) {
            return nil, nil // empty result → all keys get ErrNotFound
        },
        dataloader.Config{BatchInterval: 16 * time.Millisecond, BatchSize: 5},
    )
    defer stop()

    v, err := dl.Load("1")
    assert.Empty(t, v)
    assert.ErrorIs(t, err, dataloader.ErrNotFound)
}

func TestDataloader_ErrCanceled(t *testing.T) {
    dl, stop := dataloader.New(ctx, batchFn,
        dataloader.Config{BatchInterval: 16 * time.Millisecond, BatchSize: 5},
    )
    stop() // cancel before loading

    v, err := dl.Load("1")
    assert.Empty(t, v)
    assert.ErrorIs(t, err, dataloader.ErrCanceled)
}
```

## Best Practices

1. **Use Short Batch Intervals**: 16 ms is a good default for web applications
2. **One Loader Per Request**: Create loaders scoped to the request lifecycle, not globally
3. **Always Call Stop**: Use `defer stop()` immediately after `New` to avoid goroutine leaks
4. **Check Per-Key Errors**: `LoadMany` never returns an outer error — always inspect `Result.Error`
5. **Context Propagation**: Pass context through all operations for proper cancellation
6. **Tune BatchSize to Your Backend**: Match your database's optimal bulk-query size

## License

MIT License. See [LICENSE](../../LICENSE) for details.
