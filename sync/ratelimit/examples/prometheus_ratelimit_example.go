package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newPrometheusRateLimiterMetrics() *ratelimit.PrometheusRateLimiterMetricsCollector {
	total := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_total_requests",
		Help: "Total rate limit requests.",
	})
	allowed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_allowed",
		Help: "Total allowed requests.",
	})
	denied := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_denied",
		Help: "Total denied requests.",
	})
	prometheus.MustRegister(total, allowed, denied)
	return &ratelimit.PrometheusRateLimiterMetricsCollector{
		TotalRequests: total,
		Allowed:       allowed,
		Denied:        denied,
	}
}

// Dummy ratelimiter that always allows

type alwaysAllow struct{}

func (a *alwaysAllow) Allow() bool       { return true }
func (a *alwaysAllow) AllowN(n int) bool { return true }

func main() {
	metrics := newPrometheusRateLimiterMetrics()
	rl := ratelimit.New(&alwaysAllow{}).WithMetricsCollector(metrics)

	// Simulate some requests
	for i := 0; i < 10; i++ {
		rl.Allow()
		rl.AllowN(2)
		time.Sleep(100 * time.Millisecond)
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus ratelimit metrics available at http://localhost:2117/metrics")
	log.Fatal(http.ListenAndServe(":2117", nil))
}
