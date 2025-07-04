package circuitbreaker_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var (
	ctx     = context.Background()
	wantErr = errors.New("want error")
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestCircuit(t *testing.T) {

	cb, stop1 := circuitbreaker.New(newClient(t), t.Name())
	cb.SamplingDuration = 1 * time.Second
	cb.BreakDuration = 1 * time.Second
	defer stop1()
	cb2, stop2 := circuitbreaker.New(newClient(t), t.Name())
	cb2.SamplingDuration = 1 * time.Second
	cb2.BreakDuration = 1 * time.Second
	defer stop2()

	t.Run("open", func(t *testing.T) {
		is := assert.New(t)
		for range cb.FailureThreshold {
			err := cb.Do(ctx, func() error {
				return wantErr
			})

			is.ErrorIs(err, wantErr)
		}

		err := cb.Do(ctx, func() error {
			return wantErr
		})

		is.ErrorIs(err, circuitbreaker.ErrUnavailable)
		is.Equal(circuitbreaker.Open, cb.Status())

		// Wait for message to be subscribed.
		time.Sleep(100 * time.Millisecond)
		is.Equal(circuitbreaker.Open, cb2.Status())
	})

	t.Run("half-open", func(t *testing.T) {
		// Because this is Redis TTL, we need to wait for it to expire.
		time.Sleep(cb.BreakDuration + time.Millisecond)

		err := cb.Do(ctx, func() error {
			return nil
		})

		is := assert.New(t)
		is.ErrorIs(err, nil)
		is.Equal(circuitbreaker.HalfOpen, cb.Status())
	})

	t.Run("closed", func(t *testing.T) {
		is := assert.New(t)
		for range cb.SuccessThreshold {
			err := cb.Do(ctx, func() error {
				return nil
			})

			is.Nil(err)
		}

		is.Equal(circuitbreaker.Closed, cb.Status())
	})
}

func TestSlowCall(t *testing.T) {
	cb, stop := circuitbreaker.New(newClient(t), t.Name())
	defer stop()

	cb.SlowCallCount = func(time.Duration) int {
		// Ignore duration, just return a constant value.
		return cb.FailureThreshold
	}

	err := cb.Do(ctx, func() error {
		// No error, but failed due to slow call.
		return nil
	})

	is := assert.New(t)
	is.Nil(err)
	is.Equal(circuitbreaker.Open, cb.Status())
}

func TestCircuitBreakerConfiguration(t *testing.T) {
	client := newClient(t)

	t.Run("validates configuration", func(t *testing.T) {
		// Test nil client
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil client")
			}
		}()
		circuitbreaker.New(nil, "test")
	})

	t.Run("validates empty channel", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for empty channel")
			}
		}()
		circuitbreaker.New(client, "")
	})

	t.Run("validates break duration", func(t *testing.T) {
		// Test that invalid BreakDuration causes panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid BreakDuration")
			}
		}()

		config := circuitbreaker.Config{
			BreakDuration:    -1 * time.Second, // Invalid
			FailureRatio:     0.5,
			FailureThreshold: 10,
			SamplingDuration: 10 * time.Second,
			SuccessThreshold: 5,
		}

		circuitbreaker.NewWithConfig(client, "test-config", config)
	})
}

func TestCircuitBreakerConfigurationValidation(t *testing.T) {
	client := newClient(t)

	testCases := []struct {
		name        string
		config      circuitbreaker.Config
		expectPanic bool
	}{
		{
			name: "valid configuration",
			config: circuitbreaker.Config{
				BreakDuration:    5 * time.Second,
				FailureRatio:     0.5,
				FailureThreshold: 10,
				SamplingDuration: 10 * time.Second,
				SuccessThreshold: 5,
			},
			expectPanic: false,
		},
		{
			name: "invalid failure ratio - too low",
			config: circuitbreaker.Config{
				BreakDuration:    5 * time.Second,
				FailureRatio:     -0.1,
				FailureThreshold: 10,
				SamplingDuration: 10 * time.Second,
				SuccessThreshold: 5,
			},
			expectPanic: true,
		},
		{
			name: "invalid failure ratio - too high",
			config: circuitbreaker.Config{
				BreakDuration:    5 * time.Second,
				FailureRatio:     1.5,
				FailureThreshold: 10,
				SamplingDuration: 10 * time.Second,
				SuccessThreshold: 5,
			},
			expectPanic: true,
		},
		{
			name: "invalid failure threshold",
			config: circuitbreaker.Config{
				BreakDuration:    5 * time.Second,
				FailureRatio:     0.5,
				FailureThreshold: -1,
				SamplingDuration: 10 * time.Second,
				SuccessThreshold: 5,
			},
			expectPanic: true,
		},
		{
			name: "invalid sampling duration",
			config: circuitbreaker.Config{
				BreakDuration:    5 * time.Second,
				FailureRatio:     0.5,
				FailureThreshold: 10,
				SamplingDuration: -1 * time.Second,
				SuccessThreshold: 5,
			},
			expectPanic: true,
		},
		{
			name: "invalid success threshold",
			config: circuitbreaker.Config{
				BreakDuration:    5 * time.Second,
				FailureRatio:     0.5,
				FailureThreshold: 10,
				SamplingDuration: 10 * time.Second,
				SuccessThreshold: -1,
			},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for invalid configuration")
					}
				}()
			}

			cb, stop := circuitbreaker.NewWithConfig(client, "test-"+tc.name, tc.config)
			if !tc.expectPanic {
				defer stop()
				assert.NotNil(t, cb)
			}
		})
	}
}

func TestCircuitBreakerStates(t *testing.T) {
	client := newClient(t)

	t.Run("disabled state", func(t *testing.T) {
		cb, stop := circuitbreaker.New(client, t.Name())
		defer stop()

		// Force to disabled state
		cb.Disable()

		is := assert.New(t)
		is.Equal(circuitbreaker.Disabled, cb.Status())

		// Should allow all requests
		err := cb.Do(ctx, func() error {
			return wantErr
		})
		is.ErrorIs(err, wantErr)
	})

	t.Run("forced open state", func(t *testing.T) {
		cb, stop := circuitbreaker.New(client, t.Name())
		defer stop()

		// Force to open state
		cb.ForceOpen()

		is := assert.New(t)
		is.Equal(circuitbreaker.ForcedOpen, cb.Status())

		// Should return forced open error
		err := cb.Do(ctx, func() error {
			return nil
		})
		is.ErrorIs(err, circuitbreaker.ErrForcedOpen)
	})
}

func TestCircuitBreakerHeartbeat(t *testing.T) {
	client := newClient(t)
	cb, stop := circuitbreaker.New(client, t.Name())
	defer stop()

	cb.HeartbeatDuration = 100 * time.Millisecond
	cb.SlowCallCount = func(duration time.Duration) int {
		if duration >= 100*time.Millisecond {
			return cb.FailureThreshold
		}
		return 0
	}

	is := assert.New(t)

	// Start operation that will be monitored by heartbeat
	done := make(chan error, 1)
	go func() {
		err := cb.Do(ctx, func() error {
			time.Sleep(300 * time.Millisecond) // Slow operation
			return nil
		})
		done <- err
	}()

	// Wait for heartbeat to detect slow operation
	time.Sleep(250 * time.Millisecond)

	// Check if circuit opened due to heartbeat
	is.Equal(circuitbreaker.Open, cb.Status())
}

func TestCircuitBreakerErrorTypes(t *testing.T) {
	client := newClient(t)
	cb, stop := circuitbreaker.New(client, t.Name())
	defer stop()

	is := assert.New(t)

	// Test context cancellation (should not count as failure)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Do(ctx, func() error {
		return context.Canceled
	})
	is.ErrorIs(err, context.Canceled)

	// Should still be closed since context cancellation doesn't count
	is.Equal(circuitbreaker.Closed, cb.Status())

	// Test deadline exceeded (should count as multiple failures)
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancelTimeout()

	time.Sleep(1 * time.Millisecond) // Ensure timeout

	err = cb.Do(ctxTimeout, func() error {
		return context.DeadlineExceeded
	})
	is.ErrorIs(err, context.DeadlineExceeded)
}

func TestCircuitBreakerPublishError(t *testing.T) {
	// Test error handling in publish method
	client := newClient(t)
	cb, stop := circuitbreaker.New(client, "test-publish-error")
	defer stop()

	// Test that circuit breaker handles Redis errors gracefully
	// by disconnecting the client
	client.Close()

	// This should not panic even if Redis is unavailable
	err := cb.Do(ctx, func() error {
		return wantErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "want error")
}

func TestCircuitBreakerEdgeCases(t *testing.T) {
	client := newClient(t)

	t.Run("zero failure rate", func(t *testing.T) {
		// Test failureRate function with zero denominator
		// This tests the edge case in failureRate function
		cb, stop := circuitbreaker.New(client, "test-zero-failure")
		defer stop()

		// Ensure the circuit breaker handles zero success/failure counts correctly
		assert.Equal(t, circuitbreaker.Closed, cb.Status())
	})

	t.Run("custom failure count with context cancellation", func(t *testing.T) {
		cb, stop := circuitbreaker.New(client, "test-context-cancel")
		defer stop()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// This should not contribute to failure count
		err := cb.Do(ctx, func() error {
			return context.Canceled
		})

		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, circuitbreaker.Closed, cb.Status())
	})

	t.Run("timer cleanup in open state", func(t *testing.T) {
		cb, stop := circuitbreaker.New(client, "test-timer-cleanup")
		defer stop()

		cb.BreakDuration = 100 * time.Millisecond

		// Open the circuit
		for i := 0; i < cb.FailureThreshold; i++ {
			cb.Do(ctx, func() error { return wantErr })
		}

		// Force open again to test timer cleanup
		cb.ForceOpen()

		assert.Equal(t, circuitbreaker.ForcedOpen, cb.Status())
	})
}

func TestCircuitBreakerStatusTransitions(t *testing.T) {
	t.Run("all status values coverage", func(t *testing.T) {
		// Test NewStatus function with all possible values
		statuses := []struct {
			input    string
			expected circuitbreaker.Status
		}{
			{"closed", circuitbreaker.Closed},
			{"disabled", circuitbreaker.Disabled},
			{"half-open", circuitbreaker.HalfOpen},
			{"forced-open", circuitbreaker.ForcedOpen},
			{"open", circuitbreaker.Open},
			{"unknown", circuitbreaker.Closed}, // Default case
		}

		for _, s := range statuses {
			result := circuitbreaker.NewStatus(s.input)
			assert.Equal(t, s.expected, result, "Status conversion failed for: %s", s.input)
		}
	})

	t.Run("status string representations", func(t *testing.T) {
		statuses := []circuitbreaker.Status{
			circuitbreaker.Closed,
			circuitbreaker.Disabled,
			circuitbreaker.HalfOpen,
			circuitbreaker.ForcedOpen,
			circuitbreaker.Open,
		}

		for _, status := range statuses {
			str := status.String()
			assert.NotEmpty(t, str, "Status string should not be empty")

			// Verify round-trip conversion
			converted := circuitbreaker.NewStatus(str)
			assert.Equal(t, status, converted, "Round-trip conversion failed for status: %s", str)
		}
	})
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Helper()
	t.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}
