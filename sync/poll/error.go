package poll

import (
	"errors"
	"fmt"
	"sync"
)

var ErrThresholdExceeded = errors.New("poll: threshold exceeded")

type consecutiveError struct {
	mu                        sync.Mutex
	limit                     int64
	pending, success, failure int64
	percentageError           int64
}

func newConsecutiveError(limit, percentageError int64) *consecutiveError {
	return &consecutiveError{
		limit:           limit,
		percentageError: percentageError,
	}
}

func (e *consecutiveError) Ok() {
	e.mu.Lock()
	e.success++
	e.mu.Unlock()
}

func (e *consecutiveError) Err() {
	e.mu.Lock()
	e.failure++
	e.mu.Unlock()
}

func (e *consecutiveError) Allow() bool {
	e.mu.Lock()
	e.pending++
	lhs := e.pending * 100
	rhs := e.success*100 + e.failure*e.percentageError
	allow := lhs-rhs < e.limit*100
	fmt.Println(lhs - rhs)
	e.mu.Unlock()

	return allow
}

func (e *consecutiveError) Do(fn func() error) error {
	if !e.Allow() {
		return ErrThresholdExceeded
	}
	if err := fn(); err != nil {
		e.Err()

		return err
	}

	e.Ok()
	return nil
}
