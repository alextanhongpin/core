# telemetry

Connect experimental event logging and metrics to slog, Prometheus, and OpenTelemetry.

## Installation

```sh
go get github.com/alextanhongpin/core/telemetry
```

Check the containing module's `go.mod` for its required Go version.

## Usage

```go
package main

import (
	"context"
	"github.com/alextanhongpin/core/telemetry"
	"golang.org/x/exp/event"
	"golang.org/x/exp/slog"
)

func main() {
	handler, err := telemetry.NewSlogHandler(slog.Default())
	if err != nil {
		panic(err)
	}
	ctx := event.WithExporter(context.Background(), event.NewExporter(handler, nil))
	event.Log(ctx, "service started")
}
```

## Behavior and limits

NewSlogHandler adapts event logs using golang.org/x/exp/slog (the experimental slog package). NewMetricHandler accepts an OpenTelemetry meter; NewPrometheusHandler accepts a Prometheus registerer. Constructors return errors for invalid inputs; options configure error callbacks. MultiHandler forwards events through Log, Metric, then Trace and closes all handlers implementing io.Closer, returning the first close error. NewTracer installs a global tracer provider and propagator using OTLP HTTP with insecure transport; configure the collector through OTEL_EXPORTER_OTLP_ENDPOINT and call its returned shutdown function with a bounded context. Set SampleRate explicitly (0 disables sampling for new roots). HTTPHandler instruments requests except /health; NewHTTPClient provides a traced client with a 30-second timeout. NewDB opens an instrumented postgres database; register a compatible database/sql driver and close the DB. See [examples](examples/README.md) for larger integrations.
