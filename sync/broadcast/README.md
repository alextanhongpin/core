# broadcast

An in-process, typed fan-out: one `Send` delivers the value to every
subscriber, in order, through channels or callbacks.

**Use it for:** notifying several goroutines of the same events inside one
process (config reloads, state changes, internal event buses).

**Don't use it for:** lossy or buffered pub/sub where slow consumers must not
slow the sender (delivery is lossless and applies backpressure), or for
cross-process messaging. For one-to-many pipeline stages see
[`sync/pipeline`](../pipeline) (`Tee`, `FanOut`).

## Installation

```sh
go get github.com/alextanhongpin/core/sync/broadcast
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/broadcast"
)

func main() {
	b, stop := broadcast.New[int]()
	defer stop()

	ch, unsubscribe := b.Subscribe()
	defer unsubscribe()

	go b.Send(42)
	fmt.Println(<-ch)
}
```

It prints `42`.

## API

```go
func New[T any]() (*Broadcast[T], func()) // broadcast and an idempotent stop func

func (*Broadcast[T]) Send(n T)                         // blocks until the dispatcher accepts n or stop begins
func (*Broadcast[T]) Subscribe() (<-chan T, func())    // channel plus idempotent unsubscribe
func (*Broadcast[T]) Chan() <-chan T                   // Subscribe without unsubscribe; lasts until stop
func (*Broadcast[T]) Go(fn func(T))                    // run fn for each value until stop
```

## Behavior and guarantees

- **Ordering:** a single dispatcher delivers values serially, so every
  subscriber sees values in `Send` order.
- **Backpressure:** a subscriber that does not receive blocks delivery to later
  subscribers, later `Send` calls, and registrations (`Subscribe`, `Chan`, `Go`
  wait for the dispatcher). Keep receiving, or unsubscribe.
- **Unsubscribe:** idempotent and safe from any goroutine, including while a
  `Send` is blocked delivering to that subscriber. It removes the subscriber,
  closes its channel, and drops values not yet delivered to it.
- **Stop:** idempotent and concurrent-safe. It interrupts pending delivery,
  closes all subscriber channels, and waits for `Go` callbacks to return.
  Accepted values may be discarded.
- **After stop:** `Send` and `Go` do nothing; `Chan` and `Subscribe` return a
  closed channel (so `range` terminates), and the unsubscribe func is a no-op.
- **Callbacks:** `Go` callbacks must finish for stop to return and must not
  call stop, `Send`, `Chan`, `Subscribe`, or `Go` on the same instance.
- Values are shared, not copied; synchronize mutable payloads yourself.
- The zero value is not usable; construct with `New`.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
go test -count=1 ./...
```
