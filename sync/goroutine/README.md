# goroutine

Own one piece of cancelable background work, replace it, and wait for it to
finish on shutdown.

**Use it for:** a single background loop (a poller, a watcher, a refresher) that
is restarted with new settings and stopped cleanly with the service.

**Don't use it for:** fan-out of many tasks with error collection (use
`golang.org/x/sync/errgroup` or [`sync/pipeline`](../pipeline)), or work that
cannot observe `ctx` cancellation, since `Stop` would block on it.

## Installation

```sh
go get github.com/alextanhongpin/core/sync/goroutine
```

Check the module's `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/sync/goroutine"
)

func main() {
	g := goroutine.New() // or: var g goroutine.Goroutine
	started := make(chan struct{})
	stopped := make(chan struct{})
	g.Start(context.Background(), func(ctx context.Context) {
		close(started)
		<-ctx.Done() // honor cancellation
		close(stopped)
	})
	<-started
	g.Stop() // cancels and waits
	<-stopped
	fmt.Println("stopped")
}
```

It prints `stopped`.

## API

```go
func New() *Goroutine // zero value of Goroutine is also ready to use
func (*Goroutine) Start(ctx context.Context, fn func(ctx context.Context)) // cancel previous work, start fn
func (*Goroutine) Stop()                                                   // cancel current work and wait for all callbacks
```

## Behavior and guarantees

- `Start` cancels the previously started callback's context but does **not**
  wait for it, so the old and new callbacks can overlap briefly. If you need
  them not to overlap, call `Stop` first, or make the callbacks coordinate.
- `Stop` cancels the current context and waits for **every** callback started
  on this instance, including ones superseded by `Start`. It is safe to call
  repeatedly and on an unused instance.
- The callback's context derives from the `ctx` given to `Start`, and is also
  canceled when the callback returns. Parent cancellation reaches the callback.
- Callbacks must return after `ctx` is canceled. A callback that ignores
  cancellation makes `Stop` block forever.
- A callback must not call `Start` or `Stop` on the same instance while `Stop`
  is waiting (deadlock).
- Panics in a callback are not recovered and crash the process; recover inside
  the callback if needed. There is no error return; report errors through your
  own channel or state.
- Safe for concurrent use; `Start` and `Stop` calls are serialized.

## Testing

```sh
go vet ./...
go test -race -count=1 ./...
```
