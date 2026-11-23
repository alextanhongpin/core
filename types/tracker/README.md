# tracker

Log the steps and outcome of one operation with `log/slog`: shared attributes,
caller source locations, and a single success or failure record with the
elapsed time.

**Use it for:** instrumenting a request handler, job, or retry loop so its
logs share context and end with exactly one `took=` record.

**Don't use it for:** metrics or tracing (see
[`telemetry`](../../telemetry)), or as a general logger.

## Installation

```sh
go get github.com/alextanhongpin/core/types/tracker
```

This package is part of the `types` module; check its `go.mod` for the required Go version.

## Quick start

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/alextanhongpin/core/types/tracker"
)

func runJob(ctx context.Context, fail bool) error {
	t := tracker.New(ctx, slog.String("job", "sync"))
	t.Info("started")
	if fail {
		return t.Error(errors.New("upstream unavailable"))
	}
	t.Done("finished")
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	_ = runJob(context.Background(), false)
	_ = runJob(context.Background(), true)
}
```

It logs `started` and `finished took=...` for the first call, and `started`
and `upstream unavailable took=...` at error level for the second, each with
`job=sync`.

## API

```go
func New(ctx context.Context, attrs ...slog.Attr) *Tracker // uses slog.Default()
func NewWithLogger(ctx context.Context, logger *slog.Logger, attrs ...slog.Attr) *Tracker

func (t *Tracker) Debug(msg string, attrs ...slog.Attr)
func (t *Tracker) Info(msg string, attrs ...slog.Attr)
func (t *Tracker) Warn(msg string, attrs ...slog.Attr)
func (t *Tracker) Done(msg string, attrs ...slog.Attr)
func (t *Tracker) Error(err error, attrs ...slog.Attr) error
func (t *Tracker) Errorf(format string, args ...any) error
func (t *Tracker) Log(ctx context.Context, depth int, level slog.Level, msg string, attrs ...slog.Attr)
func (t *Tracker) SetAttrs(attrs ...slog.Attr)

func NewNoop() *Noop // same methods, logs nothing
```

## Behavior and guarantees

- The timer starts in `New`. `Done`, `Error`, and `Errorf` add
  `took=<elapsed>`.
- Only the first `Done` logs. Calling `Error` or `Errorf` first suppresses
  `Done`, so `defer t.Done(...)` plus `return t.Error(err)` logs one outcome.
  `Error` and `Errorf` log on every call and return the error; `Error(nil)`
  logs nothing and returns nil.
- Every record uses the tracker's context (for context-aware handlers) and the
  source location of the code that called the tracker method. With
  `defer t.Done(...)`, that location is where the deferred call runs.
- `Log` takes an explicit `runtime.Callers` depth: 2 is the caller of `Log`.
- `SetAttrs` adds attributes to all later records. It is not safe to call
  concurrently with other methods; the logging methods are safe for concurrent
  use.
- `Noop` logs nothing, but its `Error` and `Errorf` still return the error, so
  code can swap it in without swallowing failures.

## Testing

```sh
go vet ./tracker/
go test -race -count=1 ./tracker/
```

Run from the `types` directory.
