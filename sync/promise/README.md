# Promise

A lightweight Go implementation of JavaScript-style promises, built on generics and `context.Context`. Promises allow you to launch async work and await its result from multiple goroutines — the underlying function is guaranteed to run exactly once.

## Features

- **JavaScript-style API** — `New`, `Deferred`, `Resolve`, `Reject`, `WithResolvers`
- **Context-first** — every constructor takes a `context.Context` for cancellation/timeout propagation
- **Single execution** — the work function runs exactly once, regardless of how many goroutines call `Await`
- **Abort support** — cancel an in-flight promise with a specific cause via `Abort(err)`
- **Status tracking** — non-blocking `Status()` check (`Pending`, `Fulfilled`, `Rejected`)
- **Collection helpers** — `All`, `AllSettled`, `Race`, `Any` mirror the JS Promise API
- **GC-aware Map** — `Map[K, V]` stores promises under keys using weak pointers; entries are collected when no longer referenced
- **Channel primitive** — `Channel[T]` provides single-value delivery with context cancellation and one-time send/receive semantics, used internally by `Promise`
- **Generics** — fully type-safe thanks to Go generics

## Installation

```bash
go get github.com/alextanhongpin/core/sync/promise
```

> Requires **Go 1.27+** (uses `weak` package and `runtime.AddCleanup`).

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/alextanhongpin/core/sync/promise"
)

func main() {
    ctx := context.Background()

    // Launch async work
    p := promise.New(ctx, func(ctx context.Context) (string, error) {
        time.Sleep(50 * time.Millisecond)
        return "hello", nil
    })

    // Await the result (safe to call from multiple goroutines)
    result, err := p.Await()
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println(result) // hello
}
```

## API Reference

### Constructors

#### `New[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) *Promise[T]`

Creates a promise that runs `fn` in a new goroutine immediately.
The context is passed into `fn`, enabling cooperative cancellation.

```go
p := promise.New(ctx, func(ctx context.Context) (int, error) {
    select {
    case <-ctx.Done():
        return 0, context.Cause(ctx)
    case <-time.After(100 * time.Millisecond):
        return 42, nil
    }
})
result, err := p.Await()
```

#### `Deferred[T any](ctx context.Context) (*Promise[T], context.Context)`

Creates a promise that is resolved or rejected manually.
Returns both the promise and a child context that is cancelled when `Abort` is called.

```go
p, ctx := promise.Deferred[string](ctx)

go func() {
    // ... do work ...
    p.Resolve("done")
    // or: p.Reject(err)
}()

result, err := p.Await()
```

#### `Resolve[T any](ctx context.Context, v T) *Promise[T]`

Returns a promise that is already resolved with `v`.

```go
p := promise.Resolve(ctx, 42)
v, _ := p.Await() // 42
```

#### `Reject[T any](ctx context.Context, err error) *Promise[T]`

Returns a promise that is already rejected with `err`.

```go
p := promise.Reject[int](ctx, errors.ErrUnsupported)
_, err := p.Await() // errors.ErrUnsupported
```

#### `WithResolvers[T any](ctx context.Context) (p *Promise[T], resolve func(T), reject func(error))`

Returns a deferred promise together with standalone `resolve` and `reject` functions — useful when you want to pass closures without exposing the promise struct.

```go
p, resolve, reject := promise.WithResolvers[int](ctx)
go func() {
    v, err := doWork()
    if err != nil {
        reject(err)
        return
    }
    resolve(v)
}()
result, err := p.Await()
```

---

### Promise Methods

#### `Await() (T, error)`

Blocks until the promise settles and returns the value or error.
Safe to call from multiple goroutines concurrently — all callers get the same result.

#### `Resolve(v T)`

Settles the promise with a value. Only the first call takes effect; subsequent calls are no-ops.

#### `Reject(err error)`

Settles the promise with an error. Only the first call takes effect.

#### `Abort(cause error)`

Cancels the underlying context with the given cause. If the work function respects `ctx.Done()`, it will receive this cause via `context.Cause(ctx)`.

```go
p.Abort(errors.New("user cancelled"))
_, err := p.Await()
// err == "user cancelled"
```

#### `Status() Status`

Returns the current status without blocking. Returns `StatusPending` if the promise has not yet settled.

```go
switch p.Status() {
case promise.StatusPending:
    fmt.Println("still running")
case promise.StatusFulfilled:
    fmt.Println("done")
case promise.StatusRejected:
    fmt.Println("failed")
}
```

---

### Status Constants

```go
promise.StatusPending   // work in progress
promise.StatusFulfilled // resolved successfully
promise.StatusRejected  // rejected with an error
```

---

### Collection Functions

All collection functions take variadic `*Promise[T]` and block until settled.
They panic if called with zero promises.

#### `All[T any](promises ...*Promise[T]) ([]T, error)`

Waits for every promise to fulfill. Returns the ordered results, or the first error encountered.

```go
results, err := promise.All(
    promise.Resolve(ctx, 1),
    promise.Resolve(ctx, 2),
    promise.Resolve(ctx, 3),
)
// results == []int{1, 2, 3}
```

#### `AllSettled[T any](promises ...*Promise[T]) []*Result[T]`

Waits for every promise to settle (regardless of outcome). Returns one `*Result[T]` per promise, each with `Status`, `Data`, and `Error` fields.

```go
results := promise.AllSettled(
    promise.Resolve(ctx, 1),
    promise.Reject[int](ctx, errors.ErrUnsupported),
    promise.Resolve(ctx, 3),
)
// results[0].Status == StatusFulfilled, results[0].Data == 1
// results[1].Status == StatusRejected,  results[1].Error == errors.ErrUnsupported
// results[2].Status == StatusFulfilled, results[2].Data == 3
```

#### `Race[T any](promises ...*Promise[T]) (T, error)`

Returns as soon as the **first** promise settles — whether it fulfills or rejects.

```go
result, err := promise.Race(
    promise.New(ctx, slow),
    promise.New(ctx, fast),
)
```

#### `Any[T any](promises ...*Promise[T]) (T, error)`

Returns the value of the **first** promise to **fulfill**. If all promises reject, returns a joined error containing all individual errors.

```go
result, err := promise.Any(
    promise.Reject[int](ctx, err1),
    promise.New(ctx, func(ctx context.Context) (int, error) { return 42, nil }),
)
// result == 42, err == nil
```

---

### Result Type

```go
type Result[T any] struct {
    Status Status
    Data   T
    Error  error
}
```

Used by `AllSettled` to represent each settled promise.

---

### Map

`Map[K, V]` is a generic concurrent map from keys to `*Promise[V]`. It is backed by `Cache`, which stores values via weak pointers — entries are automatically evicted by the GC once all references to the underlying promise are dropped.

#### `NewMap[K comparable, V any](ctx context.Context) *Map[K, V]`

Creates a new map. The provided context is used when creating new deferred promises for each key.

```go
m := promise.NewMap[string, int](ctx)
```

#### `(*Map[K, V]).LoadOrCreate(key K) (*Promise[V], loaded bool, err error)`

Returns the existing promise for the key, or creates a new deferred one.
`loaded` is `true` if an existing (live) promise was found.

```go
p, loaded, err := m.LoadOrCreate("user:42")
if err != nil {
    return err
}
if !loaded {
    // First caller — resolve the promise
    go func() {
        user, err := fetchUser(ctx, 42)
        if err != nil {
            p.Reject(err)
            return
        }
        p.Resolve(user)
    }()
}
result, err := p.Await()
```

After all references to the promise are dropped and the GC runs, the entry is removed from the map, so subsequent calls to `LoadOrCreate` with the same key will create a fresh promise.

#### `(*Map[K, V]).LoadAndDelete(key K) (*Promise[V], bool)`

Removes and returns the promise for the key if it exists and is still live.

---

### Channel

`Channel[T]` is a low-level primitive that underpins `Promise`. It provides single-value delivery with context cancellation and guarantees that the value is sent at most once and received at most once.

#### `NewChannel[T any](ctx context.Context) *Channel[T]`

Creates a new channel bound to `ctx`.

#### `(*Channel[T]).Send(v T)`

Publishes a value. Only the first call succeeds; subsequent calls are no-ops.

#### `(*Channel[T]).Recv() (T, error)`

Blocks until a value is sent or the context is cancelled.

#### `(*Channel[T]).Close(cause error)`

Cancels the context with `cause`. If no value was sent, `Recv` returns `cause`.

#### `(*Channel[T]).Done() bool`

Reports whether `Recv` has completed.

---

## Examples

### Concurrent fetch with `All`

```go
ctx := context.Background()

fetch := func(id int) *promise.Promise[string] {
    return promise.New(ctx, func(ctx context.Context) (string, error) {
        time.Sleep(10 * time.Millisecond)
        return fmt.Sprintf("item-%d", id), nil
    })
}

results, err := promise.All(fetch(1), fetch(2), fetch(3))
if err != nil {
    log.Fatal(err)
}
fmt.Println(results) // [item-1 item-2 item-3]
```

### Deferred promise for event-driven code

```go
p, _ := promise.Deferred[string](ctx)

// Resolve when an event fires
go func() {
    event := <-eventCh
    p.Resolve(event.Payload)
}()

// Block until event arrives
payload, err := p.Await()
```

### Abort an in-flight promise

```go
p := promise.New(ctx, func(ctx context.Context) (int, error) {
    select {
    case <-ctx.Done():
        return 0, context.Cause(ctx)
    case <-time.After(10 * time.Second):
        return 42, nil
    }
})

// Cancel early
p.Abort(errors.New("timed out externally"))

_, err := p.Await()
fmt.Println(err) // timed out externally
```

### GC-aware cache with `Map`

```go
m := promise.NewMap[string, []byte](ctx)

getData := func(key string) ([]byte, error) {
    p, loaded, err := m.LoadOrCreate(key)
    if err != nil {
        return nil, err
    }
    if !loaded {
        go func() {
            data, err := expensiveFetch(key)
            if err != nil {
                p.Reject(err)
                return
            }
            p.Resolve(data)
        }()
    }
    return p.Await()
}
```

---

## Best Practices

1. **Always pass a context** — use `context.WithTimeout` or `context.WithCancel` to bound work.
2. **Respect `ctx.Done()` in your work function** — this is how `Abort` propagates cancellation.
3. **Don't `Await` on the same goroutine that resolves** — this will deadlock for deferred promises.
4. **Check `Status()` only after `Await()`** — before settlement, it always returns `StatusPending`.
5. **Use `AllSettled` when partial failures are acceptable** — `All` stops at the first error.
6. **Let the GC manage `Map` entries** — don't hold onto promise pointers longer than needed so entries can be evicted.

## License

MIT License. See [LICENSE](../../LICENSE) for details.

