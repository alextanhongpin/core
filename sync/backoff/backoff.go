// Package backoff calculates delays; callers own waiting and cancellation.
package backoff

import (
	"math"
	"math/rand/v2"
	"time"
)

// Backoff calculates a delay for an attempt index. Retry uses index 1 for its first retry.
type Backoff interface {
	At(attempts int) time.Duration
}

var (
	_ Backoff = (*Constant)(nil)
	_ Backoff = (*Exponential)(nil)
	_ Backoff = (*Linear)(nil)
)

// Constant returns a fixed delay; nonpositive periods produce zero.
// Concurrent calls are safe provided exported fields are not mutated concurrently.
type Constant struct {
	Period time.Duration
}

// NewConstant constructs a Constant delay policy.
func NewConstant(period time.Duration) *Constant {
	return &Constant{
		Period: period,
	}
}

// At calculates the delay without waiting.
func (b *Constant) At(attempts int) time.Duration {
	if b.Period <= 0 {
		return 0
	}
	return b.Period
}

// Exponential returns full jitter below min(Base * 2^index, Cap). Negative indices use zero; indices above 62 use 62. Nonpositive Base or Cap produces zero.
// Concurrent calls are safe provided exported fields are not mutated concurrently.
type Exponential struct {
	Base time.Duration
	Cap  time.Duration
}

// NewExponential constructs a Exponential delay policy.
func NewExponential(base, cap time.Duration) *Exponential {
	return &Exponential{
		Base: base,
		Cap:  cap,
	}
}

// At calculates the delay without waiting.
func (b *Exponential) At(attempts int) time.Duration {
	if b.Base <= 0 || b.Cap <= 0 {
		return 0
	}
	if attempts < 0 {
		attempts = 0
	}
	if attempts > 62 {
		attempts = 62
	}

	multiplier := math.Pow(2, float64(attempts))
	delay := time.Duration(float64(b.Base) * multiplier)
	if delay > b.Cap || delay <= 0 {
		delay = b.Cap
	}
	if delay <= 0 {
		return 0
	}
	return rand.N(delay)
}

// Linear returns Period times the index, saturated at the maximum duration. Nonpositive periods or indices produce zero.
// Concurrent calls are safe provided exported fields are not mutated concurrently.
type Linear struct {
	Period time.Duration
}

// NewLinear constructs a Linear delay policy.
func NewLinear(period time.Duration) *Linear {
	return &Linear{
		Period: period,
	}
}

// At calculates the delay without waiting.
func (b *Linear) At(attempts int) time.Duration {
	if b.Period <= 0 || attempts <= 0 {
		return 0
	}
	if time.Duration(attempts) > time.Duration(math.MaxInt64)/b.Period {
		return time.Duration(math.MaxInt64)
	}
	return b.Period * time.Duration(attempts)
}
