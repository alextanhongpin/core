# shutdown

Run cleanup callbacks concurrently at shutdown and collect their errors.

**Use it for:** graceful shutdown of a process that owns several resources
(HTTP servers, database pools, consumers) that can be closed independently.

**Don't use it for:** resources that must close in a specific order; call them
sequentially yourself.

## Installation

```sh
go get github.com/alextanhongpin/core/types/shutdown
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/types/shutdown"
)

func main() {
	var s shutdown.Shutdown
	s.Append(func(ctx context.Context) error { return nil }) // e.g. server.Shutdown
	s.Append(func(ctx context.Context) error { return errors.New("db: close failed") })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println(s.Wait(ctx))
}
```

It prints `db: close failed`.

## API

```go
type Shutdown []func(context.Context) error

func (s *Shutdown) Append(fn func(context.Context) error) // nil is ignored
func (s Shutdown) Wait(ctx context.Context) error
```

## Behavior and guarantees

- The zero value is ready to use.
- `Wait` starts every callback in its own goroutine at once, so there is no
  ordering between callbacks.
- `Wait` blocks until every callback returns, even after `ctx` is done. Each
  callback must watch `ctx` itself to honor a deadline.
- The result is `errors.Join` of the non-nil errors in registration order, or
  nil. Once `ctx` is done, errors matching `ctx.Err()` (by `errors.Is`) are
  dropped, so a timed-out shutdown reports only real failures.
- `Append` is not safe for concurrent use; register callbacks before `Wait`.

## Testing

```sh
go vet ./shutdown/
go test -race -count=1 ./shutdown/
```

Run from the `types` directory.
