# Broadcast

A lightweight, generic fan-out / broadcast primitive for Go. `Broadcast[T]` lets multiple subscribers receive every value sent by a single publisher. It is built on top of unbuffered channels and a dedicated dispatcher goroutine, with support for graceful shutdown and background listeners.

It's ideal for event buses, pub/sub within a process, and fan-out patterns where new subscribers can be added dynamically.

## Features

- **Generic / Type-Safe**: `Broadcast[T]` works with any type.
- **Dynamic Subscribers**: Add subscribers at any time with `Chan()` or `Go()`.
- **Fan-Out**: Every `Send` is delivered to all registered subscribers.
- **Graceful Shutdown**: `stop()` closes the dispatcher, waits for workers, and closes all subscriber channels.
- **Background Workers**: `Go(fn func(T))` spawns a goroutine that processes values in order.

## Installation

```bash
go get github.com/alextanhongpin/core/sync/broadcast
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/alextanhongpin/core/sync/broadcast"
)

func main() {
    b, stop := broadcast.New[string]()
    defer stop()

    // Subscriber via channel
    ch := b.Chan()
    go func() {
        for v := range ch {
            fmt.Println("channel subscriber:", v)
        }
    }()

    // Subscriber via background worker
    b.Go(func(v string) {
        fmt.Println("worker subscriber:", v)
    })

    b.Send("hello")
    b.Send("world")

    time.Sleep(100 * time.Millisecond)
}
```

Output:
```
channel subscriber: hello
worker subscriber: hello
channel subscriber: world
worker subscriber: world
```

## Usage

### Creating a Broadcast

```go
b, stop := broadcast.New[int]()
defer stop() // closes dispatcher and all subscriber channels
```

`New` starts an internal dispatcher goroutine and returns a `stop` function. Calling `stop` closes the broadcast, closes all subscriber channels, and waits for the dispatcher.

### Sending Values

```go
b.Send(42)
```

`Send` forwards the value to every registered subscriber. The send is blocking until each subscriber receives the value. If the broadcast is closed, `Send` returns immediately.

### Subscribing with a Channel

```go
ch := b.Chan()
for v := range ch {
    // handle v
}
```

`Chan()` registers a new channel and returns a receive-only channel. If the broadcast is already closed, `Chan()` returns `nil`.

### Subscribing with a Worker

```go
b.Go(func(v string) {
    // process v in a dedicated goroutine
})
```

`Go` registers a channel internally and spawns a worker via `sync.WaitGroup.Go`. The worker reads from the channel and calls `fn` for each value, exiting when the broadcast is stopped or the channel is closed.

Multiple `Go` calls are safe; each gets its own goroutine.

## API Reference

### `type Broadcast[T any]`

```go
type Broadcast[T any] struct { ... }
```

### `func New[T any]() (*Broadcast[T], func())`

Creates a new broadcast and starts the dispatcher. Returns the broadcast instance and a `stop` function.

```go
b, stop := broadcast.New[T]()
defer stop()
```

The `stop` function is safe to call once. It closes the internal `done` channel and waits for the dispatcher to finish.

### `func (b *Broadcast[T]) Send(n T)`

Broadcasts `n` to all current subscribers. Blocks until every subscriber has received the value. Returns immediately if the broadcast is closed.

### `func (b *Broadcast[T]) Chan() <-chan T`

Registers a new subscriber channel and returns it. Returns `nil` if the broadcast is already closed.

### `func (b *Broadcast[T]) Go(fn func(T))`

Registers a subscriber and starts a background goroutine that invokes `fn` for each received value. The goroutine is managed by the broadcast's `WaitGroup` and will exit on `stop`.

## Behavior Notes

- **Unbuffered channels**: Both the internal dispatch channel and subscriber channels are unbuffered by default. A slow subscriber will block the dispatcher and therefore all other subscribers for that value.
- **Fan-out ordering**: Values are delivered to subscribers in the order they were sent, but individual subscribers may process at different speeds.
- **Close semantics**: When `stop()` is called, the dispatcher closes all subscriber channels. `Chan()` will return `nil` after close.
- **Late subscribers**: Subscribers added after the first `Send` will only receive values sent after registration.

## Real-World Example

Event bus for a service:

```go
type Event struct {
    Type string
    Payload any
}

func main() {
    bus, stop := broadcast.New[Event]()
    defer stop()

    // Logger subscriber
    bus.Go(func(e Event) {
        log.Printf("event %s: %+v", e.Type, e.Payload)
    })

    // Metrics subscriber
    counter := make(map[string]int)
    bus.Go(func(e Event) {
        counter[e.Type]++
    })

    // Simulate events
    bus.Send(Event{Type: "request", Payload: "GET /"})
    bus.Send(Event{Type: "request", Payload: "GET /api"})

    // ... shutdown
    _ = counter
}
```

## Performance Considerations

- Keep subscriber handlers fast. If processing is expensive, offload to a worker pool.
- For high-throughput scenarios, wrap the broadcast with buffered channels or a bounded queue to avoid head-of-line blocking.
- `Send` blocks, so never call it from a critical path without ensuring subscribers can keep up.

## License

MIT License. See [LICENSE](../../LICENSE) for details.
