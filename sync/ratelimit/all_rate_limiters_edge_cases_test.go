package ratelimit_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

// Fixed Window Rate Limiter Edge Cases
func TestFixedWindowNewValidation(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		period    time.Duration
		wantError error
	}{
		{
			name:      "zero limit",
			limit:     0,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidFixedWindowLimit,
		},
		{
			name:      "negative limit",
			limit:     -1,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidFixedWindowLimit,
		},
		{
			name:      "zero period",
			limit:     1,
			period:    0,
			wantError: ratelimit.ErrInvalidFixedWindowPeriod,
		},
		{
			name:      "negative period",
			limit:     1,
			period:    -time.Second,
			wantError: ratelimit.ErrInvalidFixedWindowPeriod,
		},
		{
			name:      "valid parameters",
			limit:     1,
			period:    time.Second,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ratelimit.NewFixedWindow(tt.limit, tt.period)
			if err != tt.wantError {
				t.Errorf("NewFixedWindow() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestFixedWindowAllowNValidation(t *testing.T) {
	rl := ratelimit.MustNewFixedWindow(10, time.Second)

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

// Sliding Window Rate Limiter Edge Cases
func TestSlidingWindowNewValidation(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		period    time.Duration
		wantError error
	}{
		{
			name:      "zero limit",
			limit:     0,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidSlidingWindowLimit,
		},
		{
			name:      "negative limit",
			limit:     -1,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidSlidingWindowLimit,
		},
		{
			name:      "zero period",
			limit:     1,
			period:    0,
			wantError: ratelimit.ErrInvalidSlidingWindowPeriod,
		},
		{
			name:      "negative period",
			limit:     1,
			period:    -time.Second,
			wantError: ratelimit.ErrInvalidSlidingWindowPeriod,
		},
		{
			name:      "valid parameters",
			limit:     1,
			period:    time.Second,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ratelimit.NewSlidingWindow(tt.limit, tt.period)
			if err != tt.wantError {
				t.Errorf("NewSlidingWindow() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestSlidingWindowAllowNValidation(t *testing.T) {
	rl := ratelimit.MustNewSlidingWindow(10, time.Second)

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

// Multi Fixed Window Rate Limiter Edge Cases
func TestMultiFixedWindowNewValidation(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		period    time.Duration
		wantError error
	}{
		{
			name:      "zero limit",
			limit:     0,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidMultiFixedWindowLimit,
		},
		{
			name:      "negative limit",
			limit:     -1,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidMultiFixedWindowLimit,
		},
		{
			name:      "zero period",
			limit:     1,
			period:    0,
			wantError: ratelimit.ErrInvalidMultiFixedWindowPeriod,
		},
		{
			name:      "negative period",
			limit:     1,
			period:    -time.Second,
			wantError: ratelimit.ErrInvalidMultiFixedWindowPeriod,
		},
		{
			name:      "valid parameters",
			limit:     1,
			period:    time.Second,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ratelimit.NewMultiFixedWindow(tt.limit, tt.period)
			if err != tt.wantError {
				t.Errorf("NewMultiFixedWindow() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestMultiFixedWindowAllowNValidation(t *testing.T) {
	rl := ratelimit.MustNewMultiFixedWindow(10, time.Second)

	tests := []struct {
		name string
		key  string
		n    int
		want bool
	}{
		{"empty key", "", 1, false},
		{"zero n", "key1", 0, false},
		{"negative n", "key1", -1, false},
		{"valid params", "key1", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rl.AllowN(tt.key, tt.n)
			if got != tt.want {
				t.Errorf("AllowN(%s, %d) = %v, want %v", tt.key, tt.n, got, tt.want)
			}
		})
	}
}

// Multi Sliding Window Rate Limiter Edge Cases
func TestMultiSlidingWindowNewValidation(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		period    time.Duration
		wantError error
	}{
		{
			name:      "zero limit",
			limit:     0,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidMultiSlidingWindowLimit,
		},
		{
			name:      "negative limit",
			limit:     -1,
			period:    time.Second,
			wantError: ratelimit.ErrInvalidMultiSlidingWindowLimit,
		},
		{
			name:      "zero period",
			limit:     1,
			period:    0,
			wantError: ratelimit.ErrInvalidMultiSlidingWindowPeriod,
		},
		{
			name:      "negative period",
			limit:     1,
			period:    -time.Second,
			wantError: ratelimit.ErrInvalidMultiSlidingWindowPeriod,
		},
		{
			name:      "valid parameters",
			limit:     1,
			period:    time.Second,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ratelimit.NewMultiSlidingWindow(tt.limit, tt.period)
			if err != tt.wantError {
				t.Errorf("NewMultiSlidingWindow() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestMultiSlidingWindowAllowNValidation(t *testing.T) {
	rl := ratelimit.MustNewMultiSlidingWindow(10, time.Second)

	tests := []struct {
		name string
		key  string
		n    int
		want bool
	}{
		{"empty key", "", 1, false},
		{"zero n", "key1", 0, false},
		{"negative n", "key1", -1, false},
		{"valid params", "key1", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rl.AllowN(tt.key, tt.n)
			if got != tt.want {
				t.Errorf("AllowN(%s, %d) = %v, want %v", tt.key, tt.n, got, tt.want)
			}
		})
	}
}

// Multi GCRA Rate Limiter Edge Cases
func TestMultiGCRANewValidation(t *testing.T) {
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
			wantError: ratelimit.ErrInvalidMultiGCRALimit,
		},
		{
			name:      "negative limit",
			limit:     -1,
			period:    time.Second,
			burst:     0,
			wantError: ratelimit.ErrInvalidMultiGCRALimit,
		},
		{
			name:      "zero period",
			limit:     1,
			period:    0,
			burst:     0,
			wantError: ratelimit.ErrInvalidMultiGCRAPeriod,
		},
		{
			name:      "negative period",
			limit:     1,
			period:    -time.Second,
			burst:     0,
			wantError: ratelimit.ErrInvalidMultiGCRAPeriod,
		},
		{
			name:      "negative burst",
			limit:     1,
			period:    time.Second,
			burst:     -1,
			wantError: ratelimit.ErrInvalidMultiGCRABurst,
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
			_, err := ratelimit.NewMultiGCRA(tt.limit, tt.period, tt.burst)
			if err != tt.wantError {
				t.Errorf("NewMultiGCRA() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestMultiGCRAAllowNValidation(t *testing.T) {
	rl := ratelimit.MustNewMultiGCRA(10, time.Second, 0)

	tests := []struct {
		name string
		key  string
		n    int
		want bool
	}{
		{"empty key", "", 1, false},
		{"zero n", "key1", 0, false},
		{"negative n", "key1", -1, false},
		{"valid params", "key1", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rl.AllowN(tt.key, tt.n)
			if got != tt.want {
				t.Errorf("AllowN(%s, %d) = %v, want %v", tt.key, tt.n, got, tt.want)
			}
		})
	}
}

// Concurrent Access Tests for all rate limiters
func TestAllRateLimitersConcurrentAccess(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "FixedWindow concurrent access",
			fn: func() {
				rl := ratelimit.MustNewFixedWindow(100, time.Second)
				done := make(chan bool)
				for i := 0; i < 10; i++ {
					go func() {
						defer func() { done <- true }()
						for j := 0; j < 100; j++ {
							rl.Allow()
							rl.AllowN(2)
							rl.Remaining()
							rl.RetryAt()
						}
					}()
				}
				for i := 0; i < 10; i++ {
					<-done
				}
			},
		},
		{
			name: "SlidingWindow concurrent access",
			fn: func() {
				rl := ratelimit.MustNewSlidingWindow(100, time.Second)
				done := make(chan bool)
				for i := 0; i < 10; i++ {
					go func() {
						defer func() { done <- true }()
						for j := 0; j < 100; j++ {
							rl.Allow()
							rl.AllowN(2)
							rl.Remaining()
						}
					}()
				}
				for i := 0; i < 10; i++ {
					<-done
				}
			},
		},
		{
			name: "MultiFixedWindow concurrent access",
			fn: func() {
				rl := ratelimit.MustNewMultiFixedWindow(100, time.Second)
				done := make(chan bool)
				for i := 0; i < 10; i++ {
					go func(id int) {
						defer func() { done <- true }()
						key := "key" + string(rune(id))
						for j := 0; j < 100; j++ {
							rl.Allow(key)
							rl.AllowN(key, 2)
						}
					}(i)
				}
				for i := 0; i < 10; i++ {
					<-done
				}
			},
		},
		{
			name: "MultiSlidingWindow concurrent access",
			fn: func() {
				rl := ratelimit.MustNewMultiSlidingWindow(100, time.Second)
				done := make(chan bool)
				for i := 0; i < 10; i++ {
					go func(id int) {
						defer func() { done <- true }()
						key := "key" + string(rune(id))
						for j := 0; j < 100; j++ {
							rl.Allow(key)
							rl.AllowN(key, 2)
						}
					}(i)
				}
				for i := 0; i < 10; i++ {
					<-done
				}
			},
		},
		{
			name: "MultiGCRA concurrent access",
			fn: func() {
				rl := ratelimit.MustNewMultiGCRA(100, time.Second, 10)
				done := make(chan bool)
				for i := 0; i < 10; i++ {
					go func(id int) {
						defer func() { done <- true }()
						key := "key" + string(rune(id))
						for j := 0; j < 100; j++ {
							rl.Allow(key)
							rl.AllowN(key, 2)
							rl.RetryAt(key)
						}
					}(i)
				}
				for i := 0; i < 10; i++ {
					<-done
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}
