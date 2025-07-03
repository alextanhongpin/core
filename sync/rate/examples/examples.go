// Package examples demonstrates real-world usage patterns for the rate limiting library.
package examples

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/rate"
)

// HTTPMiddleware demonstrates rate limiting for HTTP requests
type HTTPMiddleware struct {
	requestRate *rate.Rate
	errorRate   *rate.Rate
	mu          sync.RWMutex
}

func NewHTTPMiddleware() *HTTPMiddleware {
	return &HTTPMiddleware{
		requestRate: rate.NewRate(time.Minute),
		errorRate:   rate.NewRate(time.Minute),
	}
}

func (m *HTTPMiddleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track request
		m.requestRate.Inc()

		// Wrap response writer to capture status
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapper, r)

		// Track errors
		if wrapper.statusCode >= 400 {
			m.errorRate.Inc()
		}

		duration := time.Since(start)
		log.Printf("%s %s - %d - %v - Rate: %.2f req/min, Errors: %.2f/min",
			r.Method, r.URL.Path, wrapper.statusCode, duration,
			m.requestRate.Count(), m.errorRate.Count())
	})
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// CircuitBreaker demonstrates circuit breaker pattern for external services
type CircuitBreaker struct {
	limiter *rate.Limiter
	name    string
}

func NewCircuitBreaker(name string, threshold float64) *CircuitBreaker {
	limiter := rate.NewLimiter(threshold)
	// Configure for more aggressive circuit breaking
	limiter.FailureToken = 1.0
	limiter.SuccessToken = 0.5

	return &CircuitBreaker{
		limiter: limiter,
		name:    name,
	}
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	err := cb.limiter.Do(func() error {
		return fn(ctx)
	})

	if err == rate.ErrLimitExceeded {
		return fmt.Errorf("circuit breaker '%s' is open", cb.name)
	}

	return err
}

func (cb *CircuitBreaker) Stats() (success, failure, total int) {
	return cb.limiter.Success(), cb.limiter.Failure(), cb.limiter.Total()
}

// ServiceMonitor demonstrates comprehensive service health monitoring
type ServiceMonitor struct {
	serviceName string
	metrics     map[string]*rate.Errors
	mu          sync.RWMutex
}

func NewServiceMonitor(serviceName string) *ServiceMonitor {
	return &ServiceMonitor{
		serviceName: serviceName,
		metrics:     make(map[string]*rate.Errors),
	}
}

func (sm *ServiceMonitor) TrackOperation(operation string, success bool) {
	sm.mu.Lock()
	if _, exists := sm.metrics[operation]; !exists {
		sm.metrics[operation] = rate.NewErrors(5 * time.Minute)
	}
	tracker := sm.metrics[operation]
	sm.mu.Unlock()

	if success {
		tracker.Success().Inc()
	} else {
		tracker.Failure().Inc()
	}
}

func (sm *ServiceMonitor) GetHealthReport() map[string]HealthMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	report := make(map[string]HealthMetrics)
	for operation, tracker := range sm.metrics {
		errorRate := tracker.Rate()
		report[operation] = HealthMetrics{
			Operation:   operation,
			SuccessRate: errorRate.Success(),
			FailureRate: errorRate.Failure(),
			ErrorRatio:  errorRate.Ratio(),
			IsHealthy:   errorRate.Ratio() < 0.1, // < 10% error rate
			TotalEvents: errorRate.Total(),
		}
	}
	return report
}

type HealthMetrics struct {
	Operation   string  `json:"operation"`
	SuccessRate float64 `json:"success_rate"`
	FailureRate float64 `json:"failure_rate"`
	ErrorRatio  float64 `json:"error_ratio"`
	IsHealthy   bool    `json:"is_healthy"`
	TotalEvents float64 `json:"total_events"`
}

// AdaptiveRateLimiter demonstrates adaptive rate limiting based on system health
type AdaptiveRateLimiter struct {
	baseLimit      float64
	currentLimit   *rate.Limiter
	systemHealth   *rate.Errors
	lastAdjustment time.Time
	mu             sync.RWMutex
}

func NewAdaptiveRateLimiter(baseLimit float64) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimit:      baseLimit,
		currentLimit:   rate.NewLimiter(baseLimit),
		systemHealth:   rate.NewErrors(time.Minute),
		lastAdjustment: time.Now(),
	}
}

func (arl *AdaptiveRateLimiter) Allow() bool {
	arl.adjustLimitIfNeeded()
	return arl.currentLimit.Allow()
}

func (arl *AdaptiveRateLimiter) RecordSuccess() {
	arl.currentLimit.Ok()
	arl.systemHealth.Success().Inc()
}

func (arl *AdaptiveRateLimiter) RecordFailure() {
	arl.currentLimit.Err()
	arl.systemHealth.Failure().Inc()
}

func (arl *AdaptiveRateLimiter) adjustLimitIfNeeded() {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	// Only adjust every 30 seconds
	if time.Since(arl.lastAdjustment) < 30*time.Second {
		return
	}

	health := arl.systemHealth.Rate()
	if health.Total() < 10 {
		// Not enough data to make decisions
		return
	}

	errorRatio := health.Ratio()
	var newLimit float64

	switch {
	case errorRatio > 0.2: // > 20% errors: reduce limit by 50%
		newLimit = arl.baseLimit * 0.5
	case errorRatio > 0.1: // > 10% errors: reduce limit by 25%
		newLimit = arl.baseLimit * 0.75
	case errorRatio < 0.05: // < 5% errors: increase limit by 25%
		newLimit = arl.baseLimit * 1.25
	default: // 5-10% errors: use base limit
		newLimit = arl.baseLimit
	}

	// Update the limiter with new limit
	arl.currentLimit = rate.NewLimiter(newLimit)
	arl.lastAdjustment = time.Now()

	log.Printf("Adaptive limiter adjusted: error_ratio=%.2f%%, new_limit=%.2f",
		errorRatio*100, newLimit)
}

func (arl *AdaptiveRateLimiter) GetCurrentLimit() float64 {
	arl.mu.RLock()
	defer arl.mu.RUnlock()
	// Note: In a real implementation, you'd want to expose the current limit
	// This is simplified for the example
	return arl.baseLimit
}
