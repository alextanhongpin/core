# shutdown

Run shutdown callbacks concurrently and wait for completion.

## Installation

```sh
go get github.com/alextanhongpin/core/types/shutdown
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"context"
	"github.com/alextanhongpin/core/types/shutdown"
)

func main() {
	var callbacks shutdown.Shutdown
	callbacks.Append(func(ctx context.Context) error { return nil })
	callbacks.Wait(context.Background())
}
```

## Behavior and limits

Wait launches callbacks in reverse slice order, but concurrent execution does not guarantee completion order. All callbacks receive the same context and must handle cancellation themselves. Wait logs errors other than errors matching ctx.Err(); it returns no error. It waits for callbacks even after the context expires.
