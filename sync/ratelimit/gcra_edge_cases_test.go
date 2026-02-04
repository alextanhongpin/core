package ratelimit_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func TestGCRANewValidation(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		period    time.Duration
		burst     int
		wantError error
	}{
		{
			name:      "zero limit",
			limit:     0,
			period:    time.Second,
			burst:     0,
			wantError: ratelimit.ErrInvalidNumber,
		},
		{
			name:      "negative limit",
			limit:     -1,
			period:    time.Second,
			burst:     0,
			wantError: ratelimit.ErrInvalidNumber,
		},
		{
			name:      "zero period",
			limit:     1,
			period:    0,
			burst:     0,
			wantError: ratelimit.ErrInvalidNumber,
		},
		{
			name:      "negative period",
			limit:     1,
			period:    -time.Second,
			burst:     0,
			wantError: ratelimit.ErrInvalidNumber,
		},
		{
			name:      "negative burst",
			limit:     1,
			period:    time.Second,
			burst:     -1,
			wantError: ratelimit.ErrInvalidNumber,
		},
		{
			name:      "valid parameters",
			limit:     1,
			period:    time.Second,
			burst:     0,
			wantError: nil,
		},
		{
			name:      "very high burst",
			limit:     1,
			period:    time.Nanosecond,
			burst:     1 << 30, // Large burst that could cause overflow
			wantError: nil,     // Should handle overflow gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ratelimit.NewGCRA(tt.limit, tt.period, tt.burst)
			if !errors.Is(err, tt.wantError) {
				t.Errorf("NewGCRA() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestGCRAAllowNValidation(t *testing.T) {
	rl := ratelimit.MustNewGCRA(10, time.Second, 0)

	tests := []struct {
		name string
		n    int
		want bool
	}{
		{"zero n", 0, false},
		{"negative n", -1, false},
		{"positive n", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rl.AllowN(tt.n)
			if got != tt.want {
				t.Errorf("AllowN(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestGCRAHighFrequency(t *testing.T) {
	// Test with very high frequency to check for overflow issues
	rl := ratelimit.MustNewGCRA(1000000, time.Second, 0)

	now := time.Now()
	for i := 0; i < 100; i++ {
		rl.Now = func() time.Time {
			return now.Add(time.Duration(i) * time.Microsecond)
		}
		rl.Allow() // Should not panic
	}
}

func TestGCRAVeryLargeBurst(t *testing.T) {
	// Test with very large burst that would normally cause overflow
	rl := ratelimit.MustNewGCRA(1, time.Second, 1<<30)

	// Should not panic and should work correctly
	now := time.Now()
	rl.Now = func() time.Time { return now }

	// Should allow many requests due to large burst
	allowed := 0
	for i := 0; i < 1000; i++ {
		if rl.Allow() {
			allowed++
		}
	}

	if allowed == 0 {
		t.Error("Expected at least some requests to be allowed with large burst")
	}
}

func TestGCRARetryAtOverflow(t *testing.T) {
	rl := ratelimit.MustNewGCRA(1, time.Nanosecond, 0)

	// Force internal state to a value that could cause overflow
	now := time.Unix(0, 1<<62) // Very large timestamp
	rl.Now = func() time.Time { return now }

	// Consume rate limit
	rl.Allow()

	// Should not panic when calculating retry time
	retryAt := rl.RetryAt()
	if retryAt.IsZero() {
		t.Error("RetryAt should return a valid time")
	}
}

func TestGCRAConcurrentAccess(t *testing.T) {
	rl := ratelimit.MustNewGCRA(100, time.Second, 10)

	// Test concurrent access to ensure no race conditions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				rl.Allow()
				rl.AllowN(2)
				rl.RetryAt()
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGCRAExtremePeriods(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		period time.Duration
	}{
		{"very short period", 1, time.Nanosecond},
		{"very long period", 1, 24 * time.Hour},
		{"microsecond period", 1000, time.Microsecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := ratelimit.NewGCRA(tt.limit, tt.period, 0)
			if err != nil {
				t.Fatalf("NewGCRA failed: %v", err)
			}

			// Should not panic
			rl.Allow()
			rl.RetryAt()
		})
	}
}
