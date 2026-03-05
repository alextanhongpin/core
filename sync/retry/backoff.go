package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

type backoff interface {
	At(i int) time.Duration
}

var (
	_ backoff = (*ConstantBackoff)(nil)
	_ backoff = (*ExponentialBackoff)(nil)
	_ backoff = (*LinearBackoff)(nil)
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
	return rand.N(min(b.Base*time.Duration(math.Pow(2, float64(attempts))), b.Cap))
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
	return b.Period * time.Duration(attempts)
}
