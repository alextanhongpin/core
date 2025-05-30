package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

type backOffPolicy interface {
	BackOff(i int) time.Duration
}

var (
	_ backOffPolicy = (*ConstantBackOff)(nil)
	_ backOffPolicy = (*ExponentialBackOff)(nil)
	_ backOffPolicy = (*LinearBackOff)(nil)
)

type ConstantBackOff struct {
	Period time.Duration
}

func NewConstantBackOff(period time.Duration) *ConstantBackOff {
	return &ConstantBackOff{
		Period: period,
	}
}

func (b *ConstantBackOff) BackOff(attempts int) time.Duration {
	return b.Period
}

type ExponentialBackOff struct {
	Base time.Duration
	Cap  time.Duration
}

func NewExponentialBackOff(base, cap time.Duration) *ExponentialBackOff {
	return &ExponentialBackOff{
		Base: base,
		Cap:  cap,
	}
}

func (b *ExponentialBackOff) BackOff(attempts int) time.Duration {
	return rand.N(min(b.Base*time.Duration(math.Pow(2, float64(attempts))), b.Cap))
}

type LinearBackOff struct {
	Period time.Duration
}

func NewLinearBackOff(period time.Duration) *LinearBackOff {
	return &LinearBackOff{
		Period: period,
	}
}

func (b *LinearBackOff) BackOff(attempts int) time.Duration {
	return b.Period * time.Duration(attempts)
}
