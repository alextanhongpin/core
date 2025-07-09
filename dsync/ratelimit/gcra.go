package ratelimit

import (
	"context"
	_ "embed"
	"time"

	redis "github.com/redis/go-redis/v9"
)

//go:embed gcra.lua
var gcraScript string

var gcra = redis.NewScript(gcraScript)

// GCRA implements the Generic Cell Rate Algorithm for smooth rate limiting.
// It provides better traffic shaping compared to fixed windows by avoiding
// burst behavior at window boundaries.
type GCRA struct {
	Now    func() time.Time
	burst  int
	client *redis.Client
	limit  int
	period int64
	metricsCollector MetricsCollector
}

// NewGCRA creates a new GCRA rate limiter.
//
// Parameters:
//   - client: Redis client for distributed coordination
//   - limit: Maximum number of requests per period
//   - period: Time period for the rate limit
//   - burst: Additional burst capacity (0 = no burst allowed)
//
// Example:
//
//	rl := NewGCRA(client, 100, time.Second, 10)  // 100 req/sec with 10 burst
func NewGCRA(client *redis.Client, limit int, period time.Duration, burst int, collectors ...MetricsCollector) *GCRA {
	var collector MetricsCollector
	if len(collectors) > 0 && collectors[0] != nil {
		collector = collectors[0]
	} else {
		collector = &AtomicMetricsCollector{}
	}
	return &GCRA{
		Now:              time.Now,
		burst:            burst,
		client:           client,
		limit:            limit,
		period:           period.Milliseconds(),
		metricsCollector: collector,
	}
}

// Allow checks if a single request is allowed for the given key.
func (g *GCRA) Allow(ctx context.Context, key string) (bool, error) {
	g.metricsCollector.IncTotalRequests()
	allowed, err := g.AllowN(ctx, key, 1)
	if err != nil {
		g.metricsCollector.IncDenied()
		return false, err
	}
	if allowed {
		g.metricsCollector.IncAllowed()
	} else {
		g.metricsCollector.IncDenied()
	}
	return allowed, nil
}

// AllowN checks if N requests are allowed for the given key.
func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error) {
	g.metricsCollector.IncTotalRequests()
	burst := g.burst
	limit := g.limit
	now := g.Now()
	period := g.period

	interval := period / int64(limit)

	keys := []string{key}
	argv := []any{
		burst,
		interval,
		now.UnixMilli(),
		period,
		n,
	}
	ok, err := gcra.Run(ctx, g.client, keys, argv...).Int()
	if err != nil {
		g.metricsCollector.IncDenied()
		return false, err
	}
	if ok == 1 {
		g.metricsCollector.IncAllowed()
	} else {
		g.metricsCollector.IncDenied()
	}
	return ok == 1, nil
}

// Check performs a rate limit check and returns detailed information.
func (g *GCRA) Check(ctx context.Context, key string) (*Result, error) {
	return g.CheckN(ctx, key, 1)
}

// CheckN performs a rate limit check for N requests and returns detailed information.
func (g *GCRA) CheckN(ctx context.Context, key string, n int) (*Result, error) {
	allowed, err := g.AllowN(ctx, key, n)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Allowed:    allowed,
		Remaining:  -1, // GCRA doesn't have a simple "remaining" concept
		ResetAfter: 0,  // GCRA doesn't have a fixed reset time
	}

	if !allowed {
		// Calculate retry after based on the interval
		interval := time.Duration(g.period/int64(g.limit)) * time.Millisecond
		result.RetryAfter = interval * time.Duration(n)
	}

	return result, nil
}

// Remaining is not applicable for GCRA as it doesn't use fixed windows.
// This method is provided for interface compatibility and always returns -1.
func (g *GCRA) Remaining(ctx context.Context, key string) (int, error) {
	return -1, nil
}

// ResetAfter is not applicable for GCRA as it doesn't use fixed windows.
// This method is provided for interface compatibility and always returns 0.
func (g *GCRA) ResetAfter(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

// Example Prometheus integration
//
// import (
//   "github.com/prometheus/client_golang/prometheus"
//   "github.com/prometheus/client_golang/prometheus/promhttp"
//   "github.com/redis/go-redis/v9"
//   "github.com/alextanhongpin/core/dsync/ratelimit"
//   "net/http"
// )
//
// func main() {
//   totalRequests := prometheus.NewCounter(prometheus.CounterOpts{Name: "gcra_total_requests", Help: "Total GCRA requests."})
//   allowed := prometheus.NewCounter(prometheus.CounterOpts{Name: "gcra_allowed", Help: "Allowed requests."})
//   denied := prometheus.NewCounter(prometheus.CounterOpts{Name: "gcra_denied", Help: "Denied requests."})
//   prometheus.MustRegister(totalRequests, allowed, denied)
//
//   metrics := &ratelimit.PrometheusMetricsCollector{
//     TotalRequests: totalRequests,
//     Allowed: allowed,
//     Denied: denied,
//   }
//   rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//   rl := ratelimit.NewGCRA(rdb, 100, time.Second, 10, metrics)
//   http.Handle("/metrics", promhttp.Handler())
//   http.ListenAndServe(":8080", nil)
// }
