package rate

import (
	"errors"
	"sync"
)

// ErrLimitExceeded is returned when the rate limiter blocks an operation.
var ErrLimitExceeded = errors.New("rate: limit exceeded")

// Limiter implements a token-based rate limiter.
// It accumulates failure tokens on errors and subtracts success tokens on success.
// Operations are blocked when the accumulated tokens reach the limit.
//
// This is useful for implementing circuit breaker patterns where you want to
// stop operations after accumulated failures reach a threshold.
type Limiter struct {
	mu            sync.RWMutex
	limit         float64
	currentTokens float64 // Current accumulated failure tokens
	successCount  int     // Total number of successful operations
	failureCount  int     // Total number of failed operations
	FailureToken  float64 // Tokens added per failure (default: 1.0)
	SuccessToken  float64 // Tokens subtracted per success (default: 0.5)
}

// NewLimiter creates a new rate limiter with the specified token limit.
// The default configuration adds 1.0 tokens per failure and subtracts 0.5 tokens per success.
// Panics if limit is not positive.
func NewLimiter(limit float64) *Limiter {
	if limit <= 0 {
		panic("rate: limit must be positive")
	}
	return &Limiter{
		limit:        limit,
		SuccessToken: 0.5,
		FailureToken: 1.0,
	}
}

func (l *Limiter) Success() int {
	l.mu.RLock()
	n := l.successCount
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Total() int {
	l.mu.RLock()
	n := l.failureCount + l.successCount
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Failure() int {
	l.mu.RLock()
	n := l.failureCount
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Err() {
	l.mu.Lock()
	l.currentTokens = min(l.currentTokens+l.FailureToken, l.limit)
	l.failureCount++
	l.mu.Unlock()
}

func (l *Limiter) Ok() {
	l.mu.Lock()
	l.currentTokens = max(l.currentTokens-l.SuccessToken, 0)
	l.successCount++
	l.mu.Unlock()
}

func (l *Limiter) Allow() bool {
	l.mu.RLock()
	ok := l.currentTokens < l.limit
	l.mu.RUnlock()
	return ok
}

func (l *Limiter) Do(fn func() error) error {
	// Check limit while holding lock to prevent race condition
	l.mu.Lock()
	if l.currentTokens >= l.limit {
		l.mu.Unlock()
		return ErrLimitExceeded
	}
	l.mu.Unlock()

	if err := fn(); err != nil {
		l.Err()
		return err
	}

	l.Ok()
	return nil
}
