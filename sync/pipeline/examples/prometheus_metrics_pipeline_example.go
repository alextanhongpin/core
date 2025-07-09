package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/pipeline"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusPipelineMetrics() *pipeline.PrometheusPipelineMetricsCollector {
	processed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_processed_count",
		Help: "Total items processed.",
	})
	errors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_error_count",
		Help: "Total errors encountered.",
	})
	panics := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_panic_count",
		Help: "Total panics recovered.",
	})
	start := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pipeline_start_time_unix",
		Help: "Pipeline start time (unix).",
	})
	duration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pipeline_duration_seconds",
		Help: "Total execution time (seconds).",
	})
	throughput := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pipeline_throughput_rate",
		Help: "Items per second.",
	})
	errorRate := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pipeline_error_rate",
		Help: "Error rate (0-1).",
	})

	prometheus.MustRegister(processed, errors, panics, start, duration, throughput, errorRate)

	return &pipeline.PrometheusPipelineMetricsCollector{
		ProcessedCount: processed,
		ErrorCount:     errors,
		PanicCount:     panics,
		StartTime:      start,
		Duration:       duration,
		ThroughputRate: throughput,
		ErrorRate:      errorRate,
	}
}

func main() {
	metrics := newPrometheusPipelineMetrics()
	metrics.SetStartTime(time.Now())

	in := make(chan int)
	go func() {
		for i := 0; i < 100; i++ {
			in <- i
		}
		close(in)
	}()

	out := pipeline.Tap(in, func(v int) {
		metrics.IncProcessedCount()
		if v%10 == 0 {
			metrics.IncErrorCount()
		}
		if v == 42 {
			metrics.IncPanicCount()
		}
	})

	start := time.Now()
	var processed, errors int64
	for v := range out {
		processed++
		if v%10 == 0 {
			errors++
		}
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
	dur := time.Since(start)
	metrics.SetDuration(dur)
	metrics.SetThroughputRate(float64(processed) / dur.Seconds())
	metrics.SetErrorRate(float64(errors) / float64(processed))

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("Prometheus metrics available at http://localhost:2114/metrics")
		log.Fatal(http.ListenAndServe(":2114", nil))
	}()

	fmt.Println("Pipeline complete. Metrics are being served on /metrics.")
	select {}
}
