package ratelimit

type ratelimiter interface {
	Allow() bool
	AllowN(int) bool
}

type RateLimiter struct {
	rls              []ratelimiter
	metricsCollector MetricsCollector
}

func New(rls ...ratelimiter) *RateLimiter {
	return &RateLimiter{
		rls:              rls,
		metricsCollector: &AtomicMetricsCollector{},
	}
}

func (r *RateLimiter) WithMetricsCollector(collector MetricsCollector) *RateLimiter {
	if collector != nil {
		r.metricsCollector = collector
	}
	return r
}

func (r *RateLimiter) Allow() bool {
	r.metricsCollector.IncTotalRequests()
	allow := true
	for _, rl := range r.rls {
		if !rl.Allow() {
			allow = false
		}
	}
	if allow {
		r.metricsCollector.IncAllowed()
	} else {
		r.metricsCollector.IncDenied()
	}
	return allow
}

func (r *RateLimiter) AllowN(n int) bool {
	r.metricsCollector.IncTotalRequests()
	allow := true
	for _, rl := range r.rls {
		if !rl.AllowN(n) {
			allow = false
		}
	}
	if allow {
		r.metricsCollector.IncAllowed()
	} else {
		r.metricsCollector.IncDenied()
	}
	return allow
}

// Removed PrometheusGCRAMetricsCollector and its methods. Use the shared implementation in metrics.go instead.
