package retry

import (
	"github.com/alextanhongpin/core/sync/backoff"
	"time"
)

// Backoff is an alias for backoff.Backoff.
// Deprecated: use backoff.Backoff.
type Backoff = backoff.Backoff

// ConstantBackoff is an alias for backoff.Constant.
// Deprecated: use backoff.Constant.
type ConstantBackoff = backoff.Constant

// LinearBackoff is an alias for backoff.Linear.
// Deprecated: use backoff.Linear.
type LinearBackoff = backoff.Linear

// ExponentialBackoff is an alias for backoff.Exponential.
// Deprecated: use backoff.Exponential.
type ExponentialBackoff = backoff.Exponential

// NewConstantBackoff forwards to backoff.NewConstant.
// Deprecated: use backoff.NewConstant.
func NewConstantBackoff(period time.Duration) *ConstantBackoff {
	return backoff.NewConstant(period)
}

// NewLinearBackoff forwards to backoff.NewLinear.
// Deprecated: use backoff.NewLinear.
func NewLinearBackoff(period time.Duration) *LinearBackoff {
	return backoff.NewLinear(period)
}

// NewExponentialBackoff forwards to backoff.NewExponential.
// Deprecated: use backoff.NewExponential.
func NewExponentialBackoff(base, cap time.Duration) *ExponentialBackoff {
	return backoff.NewExponential(base, cap)
}
