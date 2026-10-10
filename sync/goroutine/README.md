# goroutine

Own cancelable background work and wait for it to finish.

## Installation

```sh
go get github.com/alextanhongpin/core/sync/goroutine
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"context"
	"github.com/alextanhongpin/core/sync/goroutine"
)

func main() {
	g := goroutine.New()
	g.Start(context.Background(), func(ctx context.Context) { <-ctx.Done() })
	g.Stop()
}
```

## Behavior and limits

The zero value works. Start cancels previous work without waiting, so callbacks can overlap. Stop cancels and waits for all callbacks. Callbacks must honor cancellation and must not call Start or Stop on the same instance while Stop waits.
