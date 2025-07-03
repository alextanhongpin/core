// Package rate provides rate limiting and rate tracking utilities.
//
// This package implements three main components:
//
//  1. Rate: An exponential decay rate counter that tracks events over time.
//     Use this when you need to measure the rate of events (e.g., requests per second).
//     The rate automatically decays over time using exponential smoothing.
//
//  2. Limiter: A token-based rate limiter that blocks operations when failure tokens
//     exceed a threshold. Use this for circuit breaker-like behavior where you want
//     to stop operations after too many failures.
//
//  3. Errors: Combines success and failure rate tracking using exponential decay.
//     Use this when you need to track error rates over time.
//
// All components are thread-safe and designed for concurrent use.
package rate

import (
	"sync"
	"time"
)

// counter defines the interface for rate counting operations.
type counter interface {
	Add(float64) float64
	Inc() float64
	Count() float64
}

var _ counter = (*Rate)(nil)

// Rate implements an exponential decay rate counter.
// It tracks the rate of events over a specified time period using
// exponential smoothing to automatically decay old measurements.
type Rate struct {
	Now    func() time.Time // Injectable time function for testing
	count  float64          // Current smoothed count
	last   int64            // Last update timestamp in nanoseconds
	mu     sync.Mutex       // Protects count and last fields
	period int64            // Time period in nanoseconds for rate calculation
}

// New creates a new Rate counter with a default period of 1 second.
func New() *Rate {
	return NewRate(time.Second)
}

// NewRate creates a new Rate counter with the specified time period.
// The period determines the time window for rate calculations.
// Panics if period is not positive.
func NewRate(period time.Duration) *Rate {
	if period <= 0 {
		panic("rate: period must be positive")
	}
	return &Rate{
		Now:    time.Now,
		period: period.Nanoseconds(),
	}
}

// Reset resets the rate counter to zero.
func (r *Rate) Reset() {
	r.mu.Lock()
	r.reset()
	r.mu.Unlock()
}

// Inc increments the counter by 1 and returns the current rate.
func (r *Rate) Inc() float64 {
	return r.Add(1)
}

// Add adds n to the counter and returns the current rate.
// The rate is calculated using exponential decay based on the time elapsed.
func (r *Rate) Add(n float64) float64 {
	r.mu.Lock()
	f := r.add(n)
	r.mu.Unlock()

	return f
}

// Count returns the current rate without adding anything.
// Note: This method updates the internal state due to time-based decay.
func (r *Rate) Count() float64 {
	return r.Add(0)
}

// Per returns the rate scaled to the specified time duration.
// For example, if the counter period is 1 second and you call Per(time.Minute),
// it returns the rate per minute.
func (r *Rate) Per(t time.Duration) float64 {
	return r.Count() * float64(t.Nanoseconds()) / float64(r.period)
}

func (r *Rate) reset() {
	r.count = 0
	r.last = 0
}

func (r *Rate) add(n float64) float64 {
	now := r.Now().UnixNano()
	ratio := 1 - float64(min(now-r.last, r.period))/float64(r.period)
	r.count = r.count*ratio + n
	r.last = now

	return r.count
}
