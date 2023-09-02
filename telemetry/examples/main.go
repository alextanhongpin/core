package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alextanhongpin/core/telemetry"
	"go.opentelemetry.io/otel/metric/metrictest"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/logfmt"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/event/otel"
	"golang.org/x/exp/slog"
)

func main() {
	mp := metrictest.NewMeterProvider()
	mh := otel.NewMetricHandler(mp.Meter("test"))

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

	got := metrictest.AsStructs(mp.MeasurementBatches)
	fmt.Printf("%#v", got)
}
