package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alextanhongpin/core/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/event"
	"golang.org/x/exp/event/eventtest"
	"golang.org/x/exp/slog"
)

// Available at endpoint /debug/vars
var success = expvar.NewInt("success.count")

func main() {
	reg := prometheus.DefaultRegisterer
	ph, err := telemetry.NewPrometheusHandler(reg)
	if err != nil {
		log.Fatalf("Failed to create prometheus handler: %v", err)
	}
	reg.MustRegister(collectors.NewExpvarCollector(map[string]*prometheus.Desc{
		"success.count": prometheus.NewDesc("success_count", "The number of success counts", nil, nil),
	}))

	opt := eventtest.ExporterOptions()
	opt.EnableNamespaces = true

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	slogHandler, err := telemetry.NewSlogHandler(logger)
	if err != nil {
		log.Fatalf("Failed to create slog handler: %v", err)
	}
	ctx := context.Background()
	ctx = event.WithExporter(ctx, event.NewExporter(&telemetry.MultiHandler{
		Metric: ph,
		Log:    slogHandler,
	}, opt))
	event.Log(ctx, "my event", event.Int64("myInt", 6))
	event.Log(ctx, "error event", event.String("myString", "some string value"))
	event.Error(ctx, "hello", errors.New("unexpected error has occured"))

	c := event.NewCounter("hits", &event.MetricOptions{Description: "Earth meteorite hits"})
	go func() {
		var count int
		for count < 5 {
			select {
			case <-time.After(1 * time.Second):
				event.Log(ctx, "counter", event.Int64("count", int64(count)))
				c.Record(ctx, 1023)
				count++
				success.Add(1)
			}
		}
	}()

	log.Println("listening to port *:8000")
	http.Handle("/", promhttp.InstrumentMetricHandler(reg, http.HandlerFunc(handler)))
	http.Handle("/metrics", promhttp.Handler())
	panic(http.ListenAndServe(":8000", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}
