package ratelimit

import (
	"errors"
	"math"
	"time"
)

// maxSafeInt is the largest integer a Lua (double) number represents exactly.
const maxSafeInt = 1<<53 - 1

// Config configures a limiter. The zero value is usable: it means
// 100 requests per minute with no extra burst.
//
// Config is copied by the constructors, so changing it afterwards has no
// effect on an existing limiter.
type Config struct {
	// Limit is the number of requests allowed per Period. Default 100.
	Limit int

	// Period is the length of the window. Must be at least 1ms.
	// Default one minute. It is truncated to whole milliseconds before it is
	// sent to Redis, so 1500*time.Microsecond behaves like 1ms; values below
	// 1ms are rejected.
	Period time.Duration

	// Burst is the number of extra requests GCRA allows to arrive at once on
	// top of the steady rate (an idle key admits Burst+1 requests instantly).
	// Default 0. FixedWindow ignores Burst.
	Burst int
}

// DefaultConfig returns the configuration used for zero fields:
// 100 requests per minute, no burst.
func DefaultConfig() Config { return Config{Limit: 100, Period: time.Minute} }

// WithDefaults returns a copy of c with zero Limit and Period replaced by the
// values from DefaultConfig.
func (c Config) WithDefaults() Config {
	d := DefaultConfig()
	if c.Limit == 0 {
		c.Limit = d.Limit
	}
	if c.Period == 0 {
		c.Period = d.Period
	}
	return c
}

// Validate reports whether c is usable as is. It does not apply defaults,
// so call WithDefaults first if c may contain zero fields. The constructors
// do both.
func (c Config) Validate() error {
	if c.Limit <= 0 || c.Burst < 0 || c.Limit > maxSafeInt || c.Burst > maxSafeInt-1 {
		return errors.New("ratelimit: invalid limit or burst")
	}
	if c.Period < time.Millisecond {
		return errors.New("ratelimit: period must be at least one millisecond")
	}
	return nil
}

func validateRequest(key string, n int) error {
	if n < 0 {
		return ErrNegative
	}
	if key == "" {
		return ErrEmptyKey
	}
	if n > maxSafeInt {
		return errors.New("ratelimit: quantity too large")
	}
	return nil
}

func (c Config) validateGCRA() error {
	if c.Period.Milliseconds()/int64(c.Limit) < 1 {
		return errors.New("ratelimit: Period/Limit must be at least one millisecond for GCRA")
	}
	if float64(c.Burst+1)*float64(c.Period.Milliseconds())/float64(c.Limit) > float64(math.MaxInt64/int64(time.Millisecond)) {
		return errors.New("ratelimit: burst duration overflows")
	}
	return nil
}
