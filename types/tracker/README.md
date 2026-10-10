# tracker

Log operation duration and contextual attributes with slog.

## Installation

```sh
go get github.com/alextanhongpin/core/types/tracker
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"context"
	"github.com/alextanhongpin/core/types/tracker"
	"log/slog"
)

func main() {
	t := tracker.New(context.Background(), slog.String("operation", "refresh"))
	defer t.Done("completed")
	t.Info("started")
}
```

## Behavior and limits

New uses slog.Default; NewWithLogger accepts a logger. Done logs took once. Error and Errorf log an error and suppress a later Done message; Error returns the supplied error. SetAttrs appends persistent attributes. NewNoop provides no-op methods, including Error and Errorf returning nil. Configure attributes before concurrent use.
