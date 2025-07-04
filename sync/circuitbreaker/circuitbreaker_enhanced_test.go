package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBreakerWithOptions(t *testing.T) {
	t.Run("custom options", func(t *testing.T) {
		opts := Options{
			BreakDuration:    1 * time.Second,
			FailureRatio:     0.3,
			FailureThreshold: 3,
			SuccessThreshold: 2,
		}

		cb := NewWithOptions(opts)

		if cb.BreakDuration != 1*time.Second {
			t.Errorf("expected BreakDuration to be 1s, got %v", cb.BreakDuration)
		}
		if cb.FailureRatio != 0.3 {
			t.Errorf("expected FailureRatio to be 0.3, got %v", cb.FailureRatio)
		}
		if cb.FailureThreshold != 3 {
			t.Errorf("expected FailureThreshold to be 3, got %v", cb.FailureThreshold)
		}
		if cb.SuccessThreshold != 2 {
			t.Errorf("expected SuccessThreshold to be 2, got %v", cb.SuccessThreshold)
		}
	})

	t.Run("default options", func(t *testing.T) {
		cb := NewWithOptions(Options{})

		if cb.BreakDuration != breakDuration {
			t.Errorf("expected default BreakDuration, got %v", cb.BreakDuration)
		}
		if cb.FailureRatio != failureRatio {
			t.Errorf("expected default FailureRatio, got %v", cb.FailureRatio)
		}
	})
}

func TestBreakerMetrics(t *testing.T) {
	cb := New()

	// Initial metrics should be zero
	metrics := cb.Metrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("expected TotalRequests to be 0, got %d", metrics.TotalRequests)
	}
	if metrics.CurrentState != "closed" {
		t.Errorf("expected CurrentState to be 'closed', got %s", metrics.CurrentState)
	}

	// Test successful request
	err := cb.Do(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	metrics = cb.Metrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("expected TotalRequests to be 1, got %d", metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != 1 {
		t.Errorf("expected SuccessfulRequests to be 1, got %d", metrics.SuccessfulRequests)
	}
	if metrics.FailedRequests != 0 {
		t.Errorf("expected FailedRequests to be 0, got %d", metrics.FailedRequests)
	}

	// Test failed request
	testErr := errors.New("test error")
	err = cb.Do(func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	metrics = cb.Metrics()
	if metrics.TotalRequests != 2 {
		t.Errorf("expected TotalRequests to be 2, got %d", metrics.TotalRequests)
	}
	if metrics.FailedRequests != 1 {
		t.Errorf("expected FailedRequests to be 1, got %d", metrics.FailedRequests)
	}
}

func TestBreakerCallbacks(t *testing.T) {
	var requestCount, successCount, failureCount, rejectCount int
	var stateChanges []string

	opts := Options{
		FailureThreshold: 2,
		FailureRatio:     0.6,
		SamplingDuration: 1 * time.Second,
		OnRequest: func() {
			requestCount++
		},
		OnSuccess: func(duration time.Duration) {
			successCount++
		},
		OnFailure: func(err error, duration time.Duration) {
			failureCount++
		},
		OnReject: func() {
			rejectCount++
		},
		OnStateChange: func(old, new Status) {
			stateChanges = append(stateChanges, old.String()+"->"+new.String())
		},
	}

	cb := NewWithOptions(opts)

	// Test successful request
	cb.Do(func() error { return nil })

	if requestCount != 1 {
		t.Errorf("expected requestCount to be 1, got %d", requestCount)
	}
	if successCount != 1 {
		t.Errorf("expected successCount to be 1, got %d", successCount)
	}

	// Test enough failed requests to open circuit (need enough failures to meet both threshold and ratio)
	for i := 0; i < 5; i++ {
		err := cb.Do(func() error { return errors.New("error") })
		// Once circuit opens, requests will be rejected
		if err == ErrBrokenCircuit {
			break
		}
	}

	if failureCount < 2 {
		t.Errorf("expected failureCount to be >= 2, got %d", failureCount)
	}

	// Circuit should be open now, test rejected request
	err := cb.Do(func() error { return nil })
	if err != ErrBrokenCircuit {
		t.Errorf("expected ErrBrokenCircuit, got %v", err)
	}

	if rejectCount == 0 {
		t.Errorf("expected rejectCount to be > 0, got %d", rejectCount)
	}

	// Verify callbacks were called appropriately
	if requestCount < 2 {
		t.Errorf("expected requestCount to be >= 2, got %d", requestCount)
	}
	if successCount != 1 {
		t.Errorf("expected successCount to be 1, got %d", successCount)
	}
	if failureCount < 2 {
		t.Errorf("expected failureCount to be >= 2, got %d", failureCount)
	}

	// The circuit should either have state changes or be in a different state
	// (depending on whether it opened during our test)
	t.Logf("State changes: %v", stateChanges)
	t.Logf("Current status: %v", cb.Status())
}

func TestBreakerRejectedRequestsMetrics(t *testing.T) {
	cb := NewWithOptions(Options{
		FailureThreshold: 1,
		FailureRatio:     0.5,
	})

	// Force circuit to open
	cb.Do(func() error { return errors.New("test error") })
	cb.Do(func() error { return errors.New("test error") })

	// Try to make request when circuit is open
	err := cb.Do(func() error { return nil })
	if err != ErrBrokenCircuit {
		t.Errorf("expected ErrBrokenCircuit, got %v", err)
	}

	metrics := cb.Metrics()
	if metrics.RejectedRequests == 0 {
		t.Errorf("expected RejectedRequests to be > 0, got %d", metrics.RejectedRequests)
	}
	if metrics.CurrentState != "open" {
		t.Errorf("expected CurrentState to be 'open', got %s", metrics.CurrentState)
	}
}

func TestBreakerContextHandling(t *testing.T) {
	cb := New()

	// Test context cancellation (should not count as failure)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := cb.Do(func() error {
		return ctx.Err()
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	metrics := cb.Metrics()
	// Context cancellation should still count as a failed request in metrics
	// but not trigger circuit breaker opening due to FailureCount function
	if metrics.FailedRequests != 1 {
		t.Errorf("expected FailedRequests to be 1, got %d", metrics.FailedRequests)
	}
}

func TestBreakerSlowCalls(t *testing.T) {
	slowCallCount := 0
	cb := NewWithOptions(Options{
		SlowCallCount: func(duration time.Duration) int {
			if duration > 100*time.Millisecond {
				slowCallCount++
				return 1
			}
			return 0
		},
		FailureThreshold: 1,
		FailureRatio:     0.0, // Only slow calls should trigger opening
	})

	// Make a slow call
	err := cb.Do(func() error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if slowCallCount == 0 {
		t.Error("expected slow call to be detected")
	}

	metrics := cb.Metrics()
	// Should still be successful but might trigger state change
	if metrics.SuccessfulRequests != 1 {
		t.Errorf("expected SuccessfulRequests to be 1, got %d", metrics.SuccessfulRequests)
	}
}

func TestBreakerStateTransitions(t *testing.T) {
	cb := NewWithOptions(Options{
		BreakDuration:    50 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
	})

	// Start in closed state
	if cb.Status() != Closed {
		t.Errorf("expected initial state to be Closed, got %v", cb.Status())
	}

	// Force to open state
	cb.Do(func() error { return errors.New("error1") })
	cb.Do(func() error { return errors.New("error2") })
	cb.Do(func() error { return errors.New("error3") })

	if cb.Status() != Open {
		t.Errorf("expected state to be Open, got %v", cb.Status())
	}

	// Wait for transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Should be half-open now, make a successful call to close
	err := cb.Do(func() error { return nil })
	if err != nil {
		t.Errorf("expected no error in half-open state, got %v", err)
	}

	// Should be closed again
	if cb.Status() != Closed {
		t.Errorf("expected state to be Closed after successful half-open call, got %v", cb.Status())
	}

	metrics := cb.Metrics()
	if metrics.StateTransitions == 0 {
		t.Errorf("expected StateTransitions to be > 0, got %d", metrics.StateTransitions)
	}
}
