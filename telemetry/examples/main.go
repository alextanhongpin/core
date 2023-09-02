package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alextanhongpin/core/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

func main() {
	meter := otel.GetMeterProvider().Meter(
		"instrumentation/package/name",             // This will appear as `otel_scope_name`.
		metric.WithInstrumentationVersion("0.0.1"), // This will appear as `otel_scope_version`.
	)
	mh := telemetry.NewMetricHandler(meter)
	//ctx = event.WithExporter(ctx, event.NewExporter(NewMetricHandler(meter), &event.ExporterOptions{}))

	log := logfmt.NewHandler(os.Stdout)

	opt := eventtest.ExporterOptions()
	opt.EnableNamespaces = true

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	ctx := context.Background()
	//_ = log
	ctx = event.WithExporter(ctx, event.NewExporter(&telemetry.MultiHandler{
		Metric: mh,
		Log:    log,
		Slog:   telemetry.NewSlogHandler(logger),
	}, opt))
	event.Log(ctx, "my event", event.Int64("myInt", 6))
	event.Log(ctx, "error event", event.String("myString", "some string value"))
	event.Error(ctx, "hello", errors.New("unexpected error has occured"))

	c := event.NewCounter("hits", &event.MetricOptions{Description: "Earth meteorite hits"})
	c.Record(ctx, 1023)

	fmt.Printf("%#v", meter)
}
