package main

import (
	"context"
	"log"
	"net/http"

	"github.com/alextanhongpin/core/sync/retry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusRetryMetrics() *retry.PrometheusRetryMetricsCollector {
	attempts := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_attempts_total",
		Help: "Total retry attempts.",
	})
	successes := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_successes_total",
		Help: "Total retry successes.",
	})
	failures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_failures_total",
		Help: "Total retry failures.",
	})
	throttles := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_throttles_total",
		Help: "Total retry throttles.",
	})
	limitExceeded := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "retry_limit_exceeded_total",
		Help: "Total retry limit exceeded events.",
	})

	prometheus.MustRegister(attempts, successes, failures, throttles, limitExceeded)

	return &retry.PrometheusRetryMetricsCollector{
		Attempts:      attempts,
		Successes:     successes,
		Failures:      failures,
		Throttles:     throttles,
		LimitExceeded: limitExceeded,
	}
}

func main() {
	metrics := newPrometheusRetryMetrics()
	r := retry.New().WithMetricsCollector(metrics)

	ctx := context.Background()
	_ = r.Do(ctx, func(ctx context.Context) error {
		return context.DeadlineExceeded // always fail for demo
	}, 3)

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus retry metrics available at http://localhost:2116/metrics")
	log.Fatal(http.ListenAndServe(":2116", nil))
}
