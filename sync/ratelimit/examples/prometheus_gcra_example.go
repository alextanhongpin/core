package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusGCRAMetrics() *ratelimit.PrometheusGCRAMetricsCollector {
	total := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gcra_total_requests",
		Help: "Total GCRA requests.",
	})
	allowed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gcra_allowed",
		Help: "Total allowed requests.",
	})
	denied := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gcra_denied",
		Help: "Total denied requests.",
	})
	prometheus.MustRegister(total, allowed, denied)
	return &ratelimit.PrometheusGCRAMetricsCollector{
		TotalRequests: total,
		Allowed:       allowed,
		Denied:        denied,
	}
}

func main() {
	metrics := newPrometheusGCRAMetrics()
	gcra, _ := ratelimit.NewGCRA(10, time.Second, 2)
	gcra.WithMetricsCollector(metrics)

	// Simulate some requests
	for i := 0; i < 20; i++ {
		gcra.Allow()
		time.Sleep(50 * time.Millisecond)
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus GCRA metrics available at http://localhost:2118/metrics")
	log.Fatal(http.ListenAndServe(":2118", nil))
}
