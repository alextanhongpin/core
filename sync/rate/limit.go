package rate

import (
	"errors"
	"sync"
)

var ErrLimitExceeded = errors.New("rate: limit exceeded")

type Limiter struct {
	mu           sync.RWMutex
	limit        float64
	total        float64
	success      int
	failure      int
	FailureToken float64
	SuccessToken float64
}

func NewLimiter(limit float64) *Limiter {
	return &Limiter{
		limit:        limit,
		SuccessToken: 0.5,
		FailureToken: 1.0,
	}
}

func (l *Limiter) Success() int {
	l.mu.RLock()
	n := l.success
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Total() int {
	l.mu.RLock()
	n := l.failure + l.success
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Failure() int {
	l.mu.RLock()
	n := l.failure
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Err() {
	l.mu.Lock()
	l.total = min(l.total+l.FailureToken, l.limit)
	l.failure++
	l.mu.Unlock()
}

func (l *Limiter) Ok() {
	l.mu.Lock()
	l.total = max(l.total-l.SuccessToken, 0)
	l.success++
	l.mu.Unlock()
}

func (l *Limiter) Allow() bool {
	l.mu.RLock()
	ok := l.total < l.limit
	l.mu.RUnlock()
	return ok
}

func (l *Limiter) Do(fn func() error) error {
	if !l.Allow() {
		return ErrLimitExceeded
	}

	if err := fn(); err != nil {
		l.Err()
		return err
	}

	l.Ok()
	return nil
}
