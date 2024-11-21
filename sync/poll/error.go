package poll

import (
	"errors"
	"sync"
)

var ErrLimitExceeded = errors.New("poll: limit exceeded")

type Limiter struct {
	mu           sync.RWMutex
	limit        int
	totalCount   float64
	successCount int
	failureCount int
}

func NewLimiter(limit int) *Limiter {
	return &Limiter{
		limit:      limit,
		totalCount: float64(limit),
	}
}

func (l *Limiter) SuccessCount() int {
	l.mu.RLock()
	n := l.successCount
	l.mu.RUnlock()
	return n
}

func (l *Limiter) FailureCount() int {
	l.mu.RLock()
	n := l.failureCount
	l.mu.RUnlock()
	return n
}

func (l *Limiter) Err() {
	l.mu.Lock()
	l.totalCount--
	l.failureCount++
	l.mu.Unlock()
}

func (l *Limiter) Ok() {
	l.mu.Lock()
	l.totalCount = min(l.totalCount+0.5, float64(l.limit))
	l.successCount++
	l.mu.Unlock()
}

func (l *Limiter) Allow() bool {
	l.mu.RLock()
	ok := l.totalCount > 0
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
