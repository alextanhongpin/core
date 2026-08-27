package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

type Backoff interface {
	At(attempts int) time.Duration
}

type backoff = Backoff

var (
	_ Backoff = (*ConstantBackoff)(nil)
	_ Backoff = (*ExponentialBackoff)(nil)
	_ Backoff = (*LinearBackoff)(nil)
)

type ConstantBackoff struct {
	Period time.Duration
}

func NewConstantBackoff(period time.Duration) *ConstantBackoff {
	return &ConstantBackoff{
		Period: period,
	}
}

func (b *ConstantBackoff) At(attempts int) time.Duration {
	if b.Period <= 0 {
		return 0
	}
	return b.Period
}

type ExponentialBackoff struct {
	Base time.Duration
	Cap  time.Duration
}

func NewExponentialBackoff(base, cap time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{
		Base: base,
		Cap:  cap,
	}
}

func (b *ExponentialBackoff) At(attempts int) time.Duration {
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

type LinearBackoff struct {
	Period time.Duration
}

func NewLinearBackoff(period time.Duration) *LinearBackoff {
	return &LinearBackoff{
		Period: period,
	}
}

func (b *LinearBackoff) At(attempts int) time.Duration {
	if b.Period <= 0 || attempts <= 0 {
		return 0
	}
	return b.Period * time.Duration(attempts)
}
