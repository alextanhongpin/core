package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/poll"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusPollMetrics() *poll.PrometheusPollMetricsCollector {
	batches := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poll_total_batches",
		Help: "Total batches processed.",
	})
	success := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poll_total_success",
		Help: "Total successful items.",
	})
	failures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poll_total_failures",
		Help: "Total failed items.",
	})
	idle := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "poll_total_idle_cycles",
		Help: "Total idle cycles.",
	})
	start := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "poll_start_time_unix",
		Help: "Poll start time (unix).",
	})
	running := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "poll_running",
		Help: "Poll running state (1=running, 0=stopped).",
	})

	prometheus.MustRegister(batches, success, failures, idle, start, running)

	return &poll.PrometheusPollMetricsCollector{
		TotalBatches:    batches,
		TotalSuccess:    success,
		TotalFailures:   failures,
		TotalIdleCycles: idle,
		StartTime:       start,
		Running:         running,
	}
}

func main() {
	metrics := newPrometheusPollMetrics()
	metrics.SetStartTime(time.Now())
	metrics.SetRunning(true)

	// Simulate metric updates
	metrics.IncTotalBatches()
	metrics.AddTotalSuccess(10)
	metrics.AddTotalFailures(2)
	metrics.IncTotalIdleCycles()

	p := poll.NewWithOptions(poll.PollOptions{})
	p.SetMetricsCollector(metrics) // inject Prometheus metrics

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus poll metrics available at http://localhost:2115/metrics")
	log.Fatal(http.ListenAndServe(":2115", nil))
}
