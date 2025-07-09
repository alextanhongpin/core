package main

import (
	"log"
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
	// Example: increment metrics manually (in real use, inject into pipeline logic)
	metrics.IncProcessedCount()
	metrics.IncErrorCount()
	metrics.IncPanicCount()
	metrics.SetStartTime(time.Now())
	metrics.SetDuration(5 * time.Second)
	metrics.SetThroughputRate(10.5)
	metrics.SetErrorRate(0.1)

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus metrics available at http://localhost:2113/metrics")
	log.Fatal(http.ListenAndServe(":2113", nil))
}
