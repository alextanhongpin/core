# telemetry

Handlers that route [`golang.org/x/exp/event`](https://pkg.go.dev/golang.org/x/exp/event)
logs and metrics to `log/slog`, OpenTelemetry and Prometheus, plus small
OpenTelemetry helpers for HTTP servers, HTTP clients and `database/sql`.

```sh
go get github.com/alextanhongpin/core/telemetry
```

## Usage

```go
logs, _ := telemetry.NewSlogHandler(slog.Default())
prom, _ := telemetry.NewPrometheusHandler(prometheus.DefaultRegisterer)
h := &telemetry.MultiHandler{Log: logs, Metric: prom}
defer h.Close()

ctx := event.WithExporter(context.Background(), event.NewExporter(h, nil))
event.Log(ctx, "service started", event.String("version", "1.2.3"))

hits := event.NewCounter("hits", &event.MetricOptions{Namespace: "app"})
hits.Record(ctx, 1, event.String("route", "/users"))
```

## Handlers

- `NewSlogHandler` writes log events to a `*slog.Logger`. A label named
  `error` raises the level to Error and marks the current span as failed.
  `trace_id`/`span_id` are added when the context carries a span, and
  `AddSource` reports the `event.Log` call site.
- `NewMetricHandler` records metric events on an OpenTelemetry `metric.Meter`:
  `Counter` → Int64Counter, `FloatGauge` → Float64Gauge,
  `DurationDistribution` → Float64Histogram in seconds (or milliseconds for
  `event.UnitMilliseconds`).
- `NewPrometheusHandler` registers one vector per metric name on first use.
  Histogram names get a `_seconds` (or `_milliseconds`) suffix and
  `UnitBytes` metrics a `_bytes` suffix. Label names are fixed by the first
  event; events with another label set, or a second metric reusing a name
  with a different type or namespace (`ErrMetricConflict`), are reported to
  the error handler and dropped. `Close` unregisters only what it registered.
- `MultiHandler` forwards to Log, Metric, then Trace, and `Close` closes each
  that implements `io.Closer`, returning the first error. For Trace, use
  `golang.org/x/exp/event/otel.NewTraceHandler`.

Errors never panic; they go to the `With*ErrorHandler` option, or `log.Printf`
by default. `event` defaults `MetricOptions.Namespace` to the caller's import
path, so set it explicitly.

## Tracing helpers

- `NewTracer` installs a global OTLP/HTTP tracer provider (insecure transport;
  configure with `OTEL_EXPORTER_OTLP_*`) and the W3C trace-context and baggage
  propagators. `SampleRate` is the root sampling ratio: 0 samples nothing.
  Call the returned shutdown function with a bounded context.
- `HTTPHandler` traces requests except `/health`; spans are named
  `METHOD PATTERN` once an `http.ServeMux` matches, else `METHOD`.
- `NewHTTPClient` returns a traced client (30s timeout) built on a clone of
  `http.DefaultTransport`.
- `NewDB` opens an instrumented `postgres` `*sql.DB`; import a driver that
  registers that name.
- `SlogAttrsFromSpanContext` appends `trace_id`/`span_id` attributes.
