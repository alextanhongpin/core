package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/background"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RunPrometheusExample demonstrates using PrometheusMetricsCollector with the background worker.
// To run: call RunPrometheusExample() from your main, or move this to its own main package.
func RunPrometheusExample() {
	// Create Prometheus metrics
	tasksQueued := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "worker_tasks_queued_total",
		Help: "Total number of tasks queued",
	})
	tasksProcessed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "worker_tasks_processed_total",
		Help: "Total number of tasks processed",
	})
	tasksRejected := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "worker_tasks_rejected_total",
		Help: "Total number of tasks rejected",
	})
	activeWorkers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "worker_active_workers",
		Help: "Current number of active workers",
	})

	// Register metrics
	prometheus.MustRegister(tasksQueued, tasksProcessed, tasksRejected, activeWorkers)

	// Create PrometheusMetricsCollector
	metrics := &background.PrometheusMetricsCollector{
		TasksQueued:    tasksQueued,
		TasksProcessed: tasksProcessed,
		TasksRejected:  tasksRejected,
		ActiveWorkers:  activeWorkers,
	}

	// Start Prometheus HTTP endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics at :2112/metrics")
		http.ListenAndServe(":2112", nil)
	}()

	// Create worker pool with Prometheus metrics
	ctx := context.Background()
	opts := background.Options{
		WorkerCount: 2,
		Metrics:     metrics,
	}
	worker, stop := background.NewWithOptions(ctx, opts, func(ctx context.Context, v int) {
		fmt.Printf("Processing task: %d\n", v)
		time.Sleep(500 * time.Millisecond)
	})
	defer stop()

	// Send some tasks
	for i := 0; i < 10; i++ {
		worker.Send(i)
	}

	time.Sleep(3 * time.Second)
}
